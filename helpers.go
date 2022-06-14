// Various helper funcions.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
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
	ex, err := os.Executable()
	if err != nil {
		LogError(err)
	}
	folder_path := filepath.Dir(ex) + "/"
	return strings.ReplaceAll(folder_path, "\\", "/")
}

func getDataFolder(date time.Time) string {
	return getAppBasePath() + "Data/" + date.Format(dateFormatString) + "/"
}

// Create a folder at a given path but do not return an error if the path alread exists.
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
		LogInfo("- Ctrl+C pressed in Terminal")
	}()
}

// Logger functions
//
// Wrapper around the standard log methods which apply color
// according to the severity.
func LogError(v ...any) {
	var color_reset, color_red string

	if runtime.GOOS != "windows" {
		color_reset = ""
		color_red = ""
	} else {
		color_reset = "\x1b[0m"
		color_red = "\x1b[31m"
	}
	logger.Fatal(color_red+"[Error]: ", fmt.Sprint(v...), color_reset)
}

func LogWarnSevere(v ...any) {
	var color_reset, color_red string

	if runtime.GOOS != "windows" {
		color_reset = ""
		color_red = ""
	} else {
		color_reset = "\x1b[0m"
		color_red = "\x1b[31m"
	}
	logger.Println(color_red+"[WarnSevere]: ", fmt.Sprint(v...), color_reset)
}

func LogWarn(v ...any) {
	var color_reset, color_yellow string

	if runtime.GOOS != "windows" {
		color_reset = ""
		color_yellow = ""
	} else {
		color_reset = "\x1b[0m"
		color_yellow = "\x1b[33m"
	}
	logger.Println(color_yellow+"[Warn]: ", fmt.Sprint(v...), color_reset)
}

func LogInfo(v ...any) {
	logger.Println("[Info]: ", fmt.Sprint(v...))
}
