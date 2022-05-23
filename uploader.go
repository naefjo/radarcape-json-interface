package main

// Upload files of the last day to the drive.
//
// Upload all the files of the previous day to google drive where a script on a local machine can
// download them and save them to the file storage.
func UploadFilesToGDrive(ticker *MidnightTicker) {
	// TODO(@naefjo): implement the logic.
	// Highlevel approach:
	// - Set up client
	// - Block until a channel signals that we are ready
	// - Iterate through all the files of the last day in the file tree
	// - Send files to drive
}
