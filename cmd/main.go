package main

import "fmt"

func Message() string {
	return "Hello from assignment 2!"
}

func main() {
	fmt.Println(Message())
}

//Comment to test the CI pipeline. This should trigger a build and run the tests.
//This will also make sure that the i as the commiter can not merge the pull request without the tests passing.
// and without the code review being approved by another team member.
//This is a good practice to ensure code quality and maintainability in a collaborative project.
//Delete this comment after testing the CI pipeline.
