# CCTA_2-4

CCTA-2.4 Assignment

This assignment is to use GET and PUT endpoints of the Canvas API.

## Get Assignment

For this assignment I elected to use the senerio of the English department wanting to check on how their faculty
were making use of "good practices" as put out by our Course Design Teams.
I had to *GET* the front page for every english course and parse it for a link to the modules section.

### Short Comings

These are things that I did not include in this assignment that would make this effort less effective if not a failure in done for real.
I did not check that the link to the modules section was for the same course that the front page was for.
This would not be a failure for this task as the data is still valid, but failing to detect an error in the link for the course will cause students to not be able to follow that link and the professor will have to fix the link after it has been reported.

## Put Assignment

For this assignment I selected the one suggested in the instructions and choose to mark every MATH 1314 course from Spring 2018 as concluded.
I had to send a PUT request to the course endpoint with the body containing json string with event set to "conclude" nested inside of a course object.

