package main

import (
	"io/ioutil"
	"time"

	"golang.org/x/sync/errgroup"
)

// Upload files of the last day to a shared drive.
//
// Upload all the files of the previous day to a shared drive where a script on a local machine can
// download them and save them to the file storage.
func UploadFilesToSharedFolder(config Config, ticker *TimeTicker) {
	logger.Println("UploadFilesToSharedFolder: Successfully started uploader goroutine.")

	for range ticker.Processor_tick_chan {
		logger.Println("UploadFilesToSharedFolder: Starting data transfer to the upload folder.")

		prev_day := time.Now().AddDate(0, 0, -1)

		// Get the data files from the last day from local storage.
		data_folder_path := getDataFolder(prev_day)
		files, err := ioutil.ReadDir(data_folder_path)
		if err != nil {
			logger.Fatal(err)
		}

		// Set up the upload folder on the shared drive.
		new_data_folder_path := config.Upload_folder_path + prev_day.Format(dateFormatString) + "/"
		err = createFolder(new_data_folder_path)
		if err != nil {
			logger.Fatal(err)
		}

		// Parallelize copying of all the files.
		// var wait_group sync.WaitGroup
		var error_group errgroup.Group

		for _, file := range files {
			// wait_group.Add(1)

			// go func(file fs.FileInfo) {
			// 	defer wait_group.Done()
			// 	moveFile(data_folder_path+file.Name(), new_data_folder_path+file.Name())
			// }(file)
			file_name := file.Name()
			error_group.Go(func() error {
				return moveFile(data_folder_path+file_name, new_data_folder_path+file_name)
			})
		}

		// Wait until all files are moved to the new folder
		// wait_group.Wait()
		// logger.Println("UploadFilesToSharedFolder: Finished data transfer.")
		if err := error_group.Wait(); err != nil {
			logger.Fatal(err)
		}
	}
}

// // Moves a file at `source` to `destination`, where both paths include the filename.
// func moveFile(source string, destination string) {
// 	err := os.Rename(source, destination)
// 	if err != nil {
// 		logger.Fatal(err)
// 	}
// }
