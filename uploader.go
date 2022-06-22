package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
)

// Upload files of the last day to a shared drive.
//
// Upload all the files of the previous day to a shared drive where a script on a local machine can
// download them and save them to the file storage.
func UploadFilesToSharedFolder(config Config, ticker *TimeTicker) {
	LogInfo("UploadFilesToSharedFolder: Successfully started uploader goroutine.")

	for range ticker.Processor_tick_chan {
		err := func() error {

			LogInfo("UploadFilesToSharedFolder: Starting data transfer to the upload folder.")

			prev_day := time.Now().AddDate(0, 0, -1)

			// Get the data files from the last day from local storage.
			data_folder_path := getDataFolder(prev_day)
			files, err := ioutil.ReadDir(data_folder_path)
			if err != nil {
				return err
			}

			// Set up the upload folder on the shared drive.
			new_data_folder_path := config.Upload_folder_path + prev_day.Format(dateFormatString) + "/"
			if err := createFolder(new_data_folder_path); err != nil {
				return err
			}

			// Parallelize copying of all the files.
			var error_group errgroup.Group

			for _, file := range files {
				file_name := file.Name()
				error_group.Go(func() error {
					return MoveFile(data_folder_path+file_name, new_data_folder_path+file_name)
				})
			}

			if err := error_group.Wait(); err != nil {
				return err
			}

			LogInfo("UploadFilesToSharedFolder: Finished data transfer.")

			// Clean up the empty folder which is left behind.
			if err := os.Remove(data_folder_path); err != nil {
				return err
			}

			return nil
		}()

		// Stop the uploader goroutine if the upload fails instead of panicking
		// and terminating the program.
		if err != nil {
			LogWarnSevere(err)
			LogWarnSevere("UploadFilesToSharedFolder: Stopping the uploader goroutine.",
				"Please upload the data files manually and restart the application.")
			break
		}

	}

}

// Copy the file from `sourcePath` path to the `backupPath` path and then move it
// from `sourcePath` path to the `uploadPath` path.
func UploadFileWithBackup(sourcePath, uploadPath, backupPath string) error {
	if err := CopyFile(sourcePath, backupPath); err != nil {
		return err
	}

	if err := MoveFile(sourcePath, uploadPath); err != nil {
		return err
	}

	return nil
}

// Move a file at `sourcePath` path to the `destPath` path.
//
// GoLang: os.Rename() give error "invalid cross-device link" for Docker container with Volumes.
// MoveFile(source, destination) will work moving file between folders
// Source: https://gist.github.com/var23rav/23ae5d0d4d830aff886c3c970b8f6c6b
func MoveFile(sourcePath, destPath string) error {
	if err := CopyFile(sourcePath, destPath); err != nil {
		return fmt.Errorf("MoveFile: failed to copy file: %s", err)
	}

	// The copy was successful, so now delete the original file
	if err := os.Remove(sourcePath); err != nil {
		return fmt.Errorf("failed removing original file: %s", err)
	}
	return nil
}

// Copy a file at `sourcePath` path to the `destPath` path.
//
// Create a copy of a file at a given location without removing the original file.
// We use this funciton to create a local backup of the data files just in case the
// data upload fails.
// Source: https://gist.github.com/var23rav/23ae5d0d4d830aff886c3c970b8f6c6b
func CopyFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("couldn't open source file: %s", err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("couldn't open dest file: %s", err)
	}
	defer outputFile.Close()

	if _, err = io.Copy(outputFile, inputFile); err != nil {
		return fmt.Errorf("writing to output file failed: %s", err)
	}

	return nil
}
