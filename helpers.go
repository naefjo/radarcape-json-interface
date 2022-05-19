// Various helper funcions.
package main

import (
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"
)

// Helper function to check whether a string is present in a slice of strings.
func IsInSlice(x string, slice []string) bool {
	for _, val := range slice {
		if x == val {
			return true
		}
	}
	return false
}

// Get the path where we save our measurements
//
// We place everything into a folder on the desktop for convenience sake.
func getAppBasePath() string {
	username, err := user.Current()
	if err != nil {
		logger.Fatal(err)
	}

	folder_path := username.HomeDir + "/Desktop/Radarcape_listener/"
	return strings.ReplaceAll(folder_path, "\\", "/")
}

// Close interrupt handler
//
// If we close the program (e.g. using Ctrl+C), this function releases
// its block which resumes execution in the main loop
func waitForCloseInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	func() {
		<-c
		logger.Println("\r- Ctrl+C pressed in Terminal")
	}()
}
