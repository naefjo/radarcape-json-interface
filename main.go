// Package radarcape_listener implements an Interface to the jetvision Radarcape.
//
// Decoded aircraft data in the form of an aircraftlist.json is accessed over HTTP
// and the relevant messages are saved to CSVs.

package main

import (
	"log"
	"os"
	"time"
)

// dateFormatString defines the date format we use.
const dateFormatString string = "20060102"

// DEBUG defines whether debug prints should be enabled or not.
var DEBUG bool = false

// Set up logger to display additional information.
var logger *log.Logger = log.New(
	os.Stderr,
	"Radarcape_listener: ",
	log.LstdFlags|log.Lshortfile,
)

// Main function (duh..)
//
// Entry point for the program. Instantiate all relevant variables and launch all
// goroutines.
func main() {
	cfg_filepath := getAppBasePath() + "radarcape_listener_config.yaml"

	config := Config{}
	config.LoadConfiguration(cfg_filepath)

	// data channel between the receiver and worker goroutine.
	aircraft_data_channel := make(chan AircraftData, 50)

	// This ticker specifies the update rate with which we poll the
	// radarcape for new data.
	ticker_2hz := time.NewTicker(500 * time.Millisecond)
	defer ticker_2hz.Stop()

	// Instantiate tickers which control csv generation and data uploading.
	midnight_ticker := NewTimeTicker(0, 0, 10)
	three_am_ticker := NewTimeTicker(3, 0, 0)

	// Instantiate reveiver goroutine.
	go GetAircraftsFromHttp(aircraft_data_channel, config, ticker_2hz)

	// Instantiate worker goroutine.
	go ProcessAircraftData(aircraft_data_channel, config, midnight_ticker)

	// Instantiate uploader goroutine if a non-empty upload path was specified.
	if config.Upload_folder_path != "" {
		go UploadFilesToSharedFolder(config, three_am_ticker)
	} else {
		LogInfo("main: 'upload_folder_path' not specified in config yaml file.",
			"Saving the data locally.")
	}

	LogInfo("main: Started the radarcape listener.")
	LogInfo(
		"main: Listening on hostname ", config.Radarcape_hostname,
		" for the following aircrafts: ", config.Icao_aircraft_types,
	)

	waitForCloseInterrupt()
	midnight_ticker.Stop()

}
