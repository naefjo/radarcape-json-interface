package main

import (
	"io/fs"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

// Upload files of the last day to the drive.
//
// Upload all the files of the previous day to google drive where a script on a local machine can
// download them and save them to the file storage.
func UploadFilesToSharedFolder(config Config, ticker *TimeTicker) {
	// TODO(@naefjo): implement the logic.
	// Highlevel approach:
	// - Set up client
	// - Block until a channel signals that we are ready
	// - Iterate through all the files of the last day in the file tree
	// - Send files to drive

	for range ticker.Processor_tick_chan {
		logger.Println("UploadFilesToSharedFolder: Starting data transfer to the upload folder.")

		curr_time := time.Now()

		prev_day := curr_time.AddDate(0, 0, -1)

		data_folder_path := getDataFolder(prev_day)

		files, err := ioutil.ReadDir(data_folder_path)

		if err != nil {
			logger.Fatal(err)
		}

		new_data_folder_path := config.Upload_folder_path + prev_day.Format(dateFormatString) + "/"

		err = createFolder(new_data_folder_path)

		if err != nil {
			logger.Fatal(err)
		}

		// Parallelize copying of all the files.
		var wait_group sync.WaitGroup

		for _, file := range files {
			wait_group.Add(1)

			go func(file fs.FileInfo) {
				defer wait_group.Done()
				moveFile(data_folder_path+file.Name(), new_data_folder_path+file.Name())
			}(file)
		}

		// Wait until all files are moved to the new folder
		wait_group.Wait()
		logger.Println("UploadFilesToSharedFolder: Finished data transfer.")
	}
}

func moveFile(source string, destination string) {

	err := os.Rename(source, destination)
	if err != nil {
		logger.Fatal(err)
	}
}
