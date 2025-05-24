package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

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
	PreviousState   string `json:"previous_state,omitempty"` // Optional field for previous state
	NewState        string `json:"new_state,omitempty"`      // Optional field for new state
}

var (
	apicfg  = APIConfig{}
	term    = "6253"
	subject = "ENGL"
	catalog = ""
)

func main() {
	fmt.Println("Running GET part of Assignment ....")
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
	engl := make(chan ResultItem, 100) // Channel to collect final results for english
	math := make(chan ResultItem, 100) // Channel to collect final results for Math

	clog := sync.WaitGroup{} // Wait for everything to finish

	// Start your engines .... it's show time
	// Looking for all the courses
	var courses []CanvasCourse // holder for the final aggregate of Courses
	// we don't have more than one request to start this part off
	fmt.Println("Getting courses ...")
	search := fmt.Sprintf("%s-%s", term, subject)
	// get all the pages of courses
	fmt.Printf("Using endpoint: %saccounts/1/courses?search_term=%s to find courses\n", apicfg.BaseURL, search) // Assignment Detail Note
	courses, err = find_canvas_courses(search)
	if err != nil {
		fmt.Println("Error getting courses:", err)
		return
	}
	totalCourses := len(courses)
	fmt.Printf("Looking throught %d courses\n", totalCourses)
	for i, course := range courses {
		ok, err := check_course(course.SisCourseID, term, subject, "") // pass "" for catalog as we want ALL ENGL courses
		if !ok || err != nil {
			fmt.Printf("Error checking course %s: %v\n", course.SisCourseID, err)
			continue
		}
		clog.Add(1) // add one more item to the clog
		fmt.Printf("Processing course %s (%d of %d)\n", course.Name, i+1, totalCourses)
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
					fmt.Printf("DEBUG: Getting Nill Erros in here some how. Error is of type %T and value is %v", check.Error, check.Value)
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
			engl <- data // send the result to the results channel
		}(course) // ship this course off for processing
	}

	// Spin up a routine to clear out the clog once it has been emptied and to close the data channel
	go func() {
		clog.Wait() // wait for all of the goroutines to finish
		close(engl) // close the results channel
	}()

	// now it's time to collect our results
	finalData := Results{
		Data: make([]ResultItem, 0), // want to make sure I don't end up with a null pointer reference again
	}
	for data := range engl {
		// This should get every returned result.
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

	report := summarize_data(finalData) // summarize the data into a report

	// Write Results to CSV File
	path := path.Join("data")
	filename := fmt.Sprintf("%s/report-%s-%s.csv", path, term, subject)
	_, err = os.Stat(filename)
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			fmt.Printf("Error creating directory %s: %v\n", path, err)
			return
		}
	}
	err = toCSVFile(report, filename)
	if err != nil {
		fmt.Printf("Error writing CSV file %s: %v\n", filename, err)
		return
	}
	fmt.Println("Completed Get Part of Assignment.")
	fmt.Println("")
	fmt.Println("Starting Put Part of Assignment ....")
	term = "6182"    // Changing to Spring 2018 term
	subject = "MATH" // Changing to Math subject
	catalog = "1314" // Setting Catalog to 1314 (College Algebra)

	clog = sync.WaitGroup{} // reset the clog for the next part

	fmt.Println("Sleeping for a minute to allow Canvas rate limit to recharge ...")
	time.Sleep(60 * time.Second) // Let's give canvas rate limit some time to recharge before we hit it with the math load
	fmt.Println(".... Moving on now...")

	// Static Check says this is not needed
	// courses = make([]CanvasCourse, 0) // reset courses for the next part
	search = fmt.Sprintf("%s-%s-%s", term, subject, catalog)
	// get all the pages of courses
	fmt.Printf("Using endpoint: %saccounts/1/courses?search_term=%s to find courses\n", apicfg.BaseURL, search) // Assignment Detail Note
	courses, err = find_canvas_courses(search)
	if err != nil {
		fmt.Println("Error getting courses:", err)
		return
	}
	totalCourses = len(courses)
	fmt.Printf("Looking throught %d courses\n", totalCourses)
	for i, course := range courses {
		ok, err := check_course(course.SisCourseID, term, subject, catalog) // passing catalog value to only get MATH 1314 courses
		if !ok || err != nil {
			fmt.Printf("Error checking course %s: %v\n", course.SisCourseID, err)
			continue
		}
		clog.Add(1) // add one more item to the clog
		fmt.Printf("Processing course %s (%d of %d)\n", course.Name, i+1, totalCourses)
		go func(course CanvasCourse) {
			defer clog.Done()        // when this function exits, remove one from the clog
			sem <- struct{}{}        // acquire a token
			defer func() { <-sem }() // release the token when done
			parts := strings.Split(course.SisCourseID, "-")
			data := ResultItem{
				CourseID:      course.ID,
				Term:          parts[0],
				Subject:       parts[2],
				Catalog:       parts[3],
				Name:          course.Name,
				Format:        course.Format,
				DefaultView:   course.DefaultView,
				PreviousState: course.WorkFlowState,
			}
			math <- data // send the result to the results channel
		}(course) // ship this course off for processing
	}

	// Spin up a routine to clear out the clog once it has been emptied and to close the data channel
	go func() {
		clog.Wait() // wait for all of the goroutines to finish
		close(math) // close the results channel
	}()

	for data := range math {
		// here is where the work happens for this part.
		// Check if the WorkFlowState is no "concluded"
		if data.PreviousState != "concluded" {
			// send PUT request to change the state to "concluded"
			fmt.Printf("Changing course %s (%d) state from %s to concluded\n", data.Name, data.CourseID, data.PreviousState)
			fmt.Printf("Using endpoint: %scourses/%d to change state to concluded\n", apicfg.BaseURL, data.CourseID) // Assignment Detail Note
			change := change_course_state(data.CourseID, "conclude")
			if change.Error != nil {
				fmt.Printf("Error changing course %s (%d) state: %v\n", data.Name, data.CourseID, change.Error)
				continue
			}
			data.NewState = change.NewState // set the new state in the data
			fmt.Printf("Changed course %s (%d) state from %s to %s\n", data.Name, data.CourseID, data.PreviousState, change.NewState)
		} else {
			fmt.Printf("Course %s (%d) is already concluded, skipping.\n", data.Name, data.CourseID)
			continue // skip this course as it is already concluded
		}
	}
	fmt.Println("Completed Put Part of Assignment.")
	fmt.Println("Assignment Completed.")
}
