package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type APIConfig struct {
	Token   string
	BaseURL string
}

type CanvasFrontPage struct {
	Body string `json:"body"`
}

type CanvasCourse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	CourseCode  string `json:"course_code"`
	SisCourseID string `json:"sis_course_id"`
	Format      string `json:"course_format"`
	Subject     string `json:"subject"`
	Catalog     string `json:"catalog"`
	DefaultView string `json:"default_view"`
}

type CheckResult struct {
	Value bool
	Error error
}

type CanvasPaginationResult struct {
	Courses []CanvasCourse
	Error   error
	NextURL *http.Request
}

func make_http_request(method string, url string, body interface{}) (*http.Request, error) {
	token := apicfg.Token
	if token == "" {
		// we have to have some sanity checks
		return nil, fmt.Errorf("invalid configuration: token empty")
	}
	var req *http.Request
	var err error
	switch b := body.(type) {
	case nil:
		req, err = http.NewRequest(method, url, nil)
	case io.Reader:
		req, err = http.NewRequest(method, url, b)
	default:
		var jsonBytes []byte
		jsonBytes, err = json.Marshal(b)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %v", err)
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonBytes))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

func get_paginated_results(req *http.Request, client http.Client) CanvasPaginationResult {
	var result CanvasPaginationResult
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to make request: %v", err)
		return result
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("request failed with status: %d (%s)", resp.StatusCode, resp.Status)
		return result
	}
	// Collect Link Header and extract Next URL
	lh := resp.Header.Get("Link")
	if lh == "" {
		fmt.Println("WARNING: No Link Header Found!!")
		fmt.Printf("Link was: %s\n", resp.Request.URL.String())
		fmt.Printf("Headers: %#v\n", resp.Header)
	}
	links := strings.Split(lh, ",")
	for _, link := range links {
		if strings.Contains(link, "rel=\"next\"") {
			parts := strings.Split(link, ";")
			if len(parts) == 2 {
				nextURL := strings.Trim(parts[0], "<>")
				result.NextURL, err = make_http_request("GET", nextURL, nil)
				if err != nil {
					result.Error = fmt.Errorf("failed to create next request: %v", err)
					return result
				}
			} else {
				result.Error = fmt.Errorf("invalid Link Header format: %s", link)
				return result
			}
		}
	}
	// Collect Body data
	result.Courses = make([]CanvasCourse, 0)
	err = json.NewDecoder(resp.Body).Decode(&result.Courses)
	if err != nil {
		result.Error = fmt.Errorf("failed to decode response body: %v", err)
		return result
	}
	return result
}

func check_for_modules_link(id int) CheckResult {
	var result CheckResult
	apipath := fmt.Sprintf("%scourses/%d/front_page", apicfg.BaseURL, id)
	req, err := make_http_request("GET", apipath, nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %v", err)
		return result
	}
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to make request: %v", err)
		return result
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("DEBUG: %s returned a non-200 status code of %d\n", req.URL.String(), resp.StatusCode)
		result.Error = fmt.Errorf("request failed with status: %d (%s)", resp.StatusCode, resp.Status)
		return result
	}
	var frontPage CanvasFrontPage
	err = json.NewDecoder(resp.Body).Decode(&frontPage)
	if err != nil {
		result.Error = fmt.Errorf("failed to decode response body: %v", err)
		return result
	}
	patt := regexp.MustCompile(`href=["'](?:https://[^"']+)?/courses/\d+/modules["']`)
	result.Value = patt.MatchString(frontPage.Body)
	result.Error = nil
	return result
}

func find_canvas_courses(search_term string) ([]CanvasCourse, error) {
	var result []CanvasCourse
	apipath := fmt.Sprintf("%saccounts/1/courses?per_page=100&search_term=%s", apicfg.BaseURL, search_term)
	req, err := make_http_request("GET", apipath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	client := http.Client{Timeout: 10 * time.Second}
	paginationResult := get_paginated_results(req, client)
	if paginationResult.Error != nil {
		return nil, paginationResult.Error
	}
	result = append(result, paginationResult.Courses...)
	for paginationResult.NextURL != nil {
		paginationResult = get_paginated_results(paginationResult.NextURL, client)
		if paginationResult.Error != nil {
			return nil, paginationResult.Error
		}
		result = append(result, paginationResult.Courses...)
	}
	return result, nil
}

func check_course(code, term, subj string) (bool, error) {
	parts := strings.Split(code, "-")
	if len(parts) != 4 {
		return false, fmt.Errorf("invalide course sis id: %s", code)
	}
	if parts[0] == term && parts[2] == subj {
		return true, nil
	}
	return false, nil
}
