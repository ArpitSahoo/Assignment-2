package main

import "testing"

// TestMessage is a test function that checks if the Message() function returns the expected string.
// This only created to test the pipeline and make sure that everything works as planned.
func TestMessage(t *testing.T) {
	got := Message()
	want := "Hello from assignment 2!"

	if got != want {
		t.Fatalf("Message() = %q, want %q", got, want)
	}

	//TODO Find out if test can be in a separate file structure than
	// under the same one
}
