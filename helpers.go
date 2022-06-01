// Various helper funcions.
package main

import (
	"errors"
	"io/fs"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"
	"time"
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

func getDataFolder(date time.Time) string {
	return getAppBasePath() + "Data/" + date.Format(dateFormatString) + "/"
}

// Create a folder at a given path but do not return an error if the paht alread exists.
func createFolder(folder_path string) error {
	err := os.MkdirAll(folder_path, os.ModePerm)

	// If we get an error apart from `Folder already Exists` we break the execution.
	if !errors.Is(err, fs.ErrExist) && err != nil {
		return err
	}

	return nil
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
