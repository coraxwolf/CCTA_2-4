package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type ReportData struct {
	Table                []ResportTableRow `json:"table" csv:"table"`
	UsedWikiCount        int               `json:"used_wiki_count" csv:"used_wiki_count"`
	FoundModuleLinkCount int               `json:"found_module_link_count" csv:"found_module_link_count"`
	UsedOtherCount       int               `json:"used_other_count" csv:"used_other_count"`
	TotalCourses         int               `json:"total_courses" csv:"total_courses"`
}

type ResportTableRow struct {
	CourseID        int    `json:"course_id" csv:"course_id"`
	Term            string `json:"term" csv:"term"`
	Subject         string `json:"subject" csv:"subject"`
	Catalog         string `json:"catalog" csv:"catalog"`
	Name            string `json:"name" csv:"name"`
	Format          string `json:"format" csv:"format"`
	DefaultView     string `json:"default_view" csv:"default_view"`
	UsedWiki        string `json:"used_wiki" csv:"used_wiki"`
	FoundModuleLink string `json:"found_module_link" csv:"found_module_link"`
}

func summarize_data(results Results) ReportData {
	report := ReportData{}

	// Capture Data Rows for table
	for _, item := range results.Data {
		row := ResportTableRow{
			CourseID:    item.CourseID,
			Term:        item.Term,
			Subject:     item.Subject,
			Catalog:     item.Catalog,
			Name:        item.Name,
			Format:      item.Format,
			DefaultView: item.DefaultView,
		}
		if item.UsedWiki {
			row.UsedWiki = "Yes"
			report.UsedWikiCount++
			if item.FoundModuleLink {
				row.FoundModuleLink = "Yes"
				report.FoundModuleLinkCount++
			}
		} else {
			row.UsedWiki = "No"
			row.FoundModuleLink = "N/A"
			report.UsedOtherCount++
		}
		report.Table = append(report.Table, row)
	}
	return report
}

func toJSONFile(data ReportData, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(data)
	if err != nil {
		fmt.Printf("Error encoding JSON: %v\n", err)
		return err
	}
	fmt.Printf("JSON file '%s' created successfully.\n", filename)
	return nil
}

func toCSVFile(data ReportData, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"course_id", "term", "subject", "catalog", "name", "format", "default_view", "used_wiki", "found_module_link"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data rows
	for _, row := range data.Table {
		record := []string{
			strconv.Itoa(row.CourseID),
			row.Term,
			row.Subject,
			row.Catalog,
			row.Name,
			row.Format,
			row.DefaultView,
			row.UsedWiki,
			row.FoundModuleLink,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	fmt.Printf("CSV file '%s' created successfully.\n", filename)
	return nil
}

func toExcelFile(data ReportData, filename string) error {
	// Placeholder for Excel file generation logic
	// This would typically use a library like "excelize" or "xlsx"
	fmt.Println("Excel file generation is not implemented yet.")
	return fmt.Errorf("function toExcelFile is not implemented yet")
}
