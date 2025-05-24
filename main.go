package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

type PageJob struct {
	Course   CanvasCourse
	Attempts int
}

type CoursePage struct {
	Courses []CanvasCourse
	Error   error
}

type Results struct {
	Data                 []ResultItem `json:"data"`
	UsedWikiCount        int          `json:"used_wiki_count"`
	FoundModuleLinkCount int          `json:"found_module_link_count"`
	UsedOtherCount       int          `json:"used_other_count"`
}

type ResultItem struct {
	CourseID        int    `json:"course_id"`
	Term            string `json:"term"`
	Subject         string `json:"subject"`
	Catalog         string `json:"catalog"`
	Name            string `json:"name"`
	Format          string `json:"format"`
	DefaultView     string `json:"default_view"`
	UsedWiki        bool   `json:"used_wiki"`
	FoundModuleLink bool   `json:"found_module_link"`
}

var (
	apicfg  = APIConfig{}
	term    = "6253"
	subject = "ENGL"
)

func main() {
	fmt.Println("Starting .....")
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}
	apicfg = APIConfig{
		Token:   os.Getenv("BETA_TOKEN"),
		BaseURL: os.Getenv("BETA_API_URL"),
	}
	// Setup channels
	sem := make(chan struct{}, 10) // Limit concurrency to 10
	// coursePages := make(chan CoursePage, 100) // Channel to collect Course Page results
	results := make(chan ResultItem, 100) // Channel to collect final results

	clog := sync.WaitGroup{} // Wait for everything to finish

	// Start your engines .... it's show time
	// Looking for all the courses
	var courses []CanvasCourse // holder for the final aggregate of Courses
	// we don't have more than one request to start this part off
	fmt.Println("Getting courses ...")
	search := fmt.Sprintf("%s-%s", term, subject)
	// get all the pages of courses
	courses, err = find_canvas_courses(search)
	if err != nil {
		fmt.Println("Error getting courses:", err)
		return
	}
	totalCourses := len(courses)
	fmt.Printf("Looking throught %d courses\n", totalCourses)
	for i, course := range courses {
		ok, err := check_course(course.SisCourseID, term, subject)
		if !ok || err != nil {
			fmt.Printf("Error checking course %s: %v\n", course.SisCourseID, err)
			continue
		}
		clog.Add(1) // add one more item to the clog
		fmt.Printf("\rProcessing course %s (%d of %d)", course.Name, i+1, totalCourses)
		go func(course CanvasCourse) {
			defer clog.Done()        // when this function exits, remove one from the clog
			sem <- struct{}{}        // acquire a token
			defer func() { <-sem }() // release the token when done
			usedWiki := course.DefaultView == "wiki"
			hasLink := false // assuming the link is missing
			if usedWiki {
				check := check_for_modules_link(course.ID)
				if check.Error != nil {
					fmt.Printf("Error checking for modules link in course %s: %v\n", course.SisCourseID, check.Error)
					// going to keep going to get the rest of the data
					hasLink = false
				}
				hasLink = check.Value // check if the link was actually there or not
			}
			parts := strings.Split(course.SisCourseID, "-")
			data := ResultItem{
				CourseID:        course.ID,
				Term:            parts[0],
				Subject:         parts[2],
				Catalog:         parts[3],
				Name:            course.Name,
				Format:          course.Format,
				DefaultView:     course.DefaultView,
				UsedWiki:        usedWiki,
				FoundModuleLink: hasLink,
			}
			results <- data // send the result to the results channel
		}(course) // ship this course off for processing
	}

	// trying the clog clean up here ... this is before the collection of the final results ....
	go func() {
		clog.Wait()    // wait for all of the goroutines to finish
		close(results) // close the results channel
	}()

	// now it's time to collect our results
	finalData := Results{
		Data: make([]ResultItem, 0), // want to make sure I don't end up with a null pointer reference again
	}
	for data := range results {
		// This should get every returned result, right?
		if data.UsedWiki {
			finalData.UsedWikiCount++
		} else {
			finalData.UsedOtherCount++
		}
		if data.FoundModuleLink {
			finalData.FoundModuleLinkCount++
		}
		finalData.Data = append(finalData.Data, data) // add result to final data Data List
	}

	// I am bound to end up here before all of the results have finished soo why does the above for loop keep working?

	// Guess I have all of the data in the finalData struct now
	// at this point just dump the data and make sure it works and didn't get cause into some endless loop again
	path := path.Join("data")
	filename := fmt.Sprintf("%s/results-%s-%s.json", path, term, subject)
	_, err = os.Stat(filename)
	// check if the file exists
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			return
		}
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close() // make sure to close the file when done
	err = json.NewEncoder(file).Encode(finalData)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
	fmt.Printf("Results written to %s\n", path)
}
