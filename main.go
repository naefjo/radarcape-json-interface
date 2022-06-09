// Package radarcape_listener implements an Interface to the jetvision Radarcape.
//
// Decoded aircraft data in the form of an aircraftlist.json is accessed over HTTP
// and the relevant messages are saved to CSVs.
//
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
var logger *log.Logger = log.New(os.Stderr, "Radarcape_listener: ", log.LstdFlags|log.Lshortfile)

// Main goroutine
//
// Entry point for the program. Instantiate all relevant variables and launch all
// goroutines.
func main() {
	cfg_filepath := getAppBasePath() + "radarcape_listener_config.yaml"

	config := Config{}
	config.LoadConfiguration(cfg_filepath)

	aircraft_data_channel := make(chan AircraftData, 50)

	ticker_2hz := time.NewTicker(500 * time.Millisecond)
	defer ticker_2hz.Stop()

	midnight_ticker := NewTimeTicker(0, 0, 10)
	three_am_ticker := NewTimeTicker(3, 0, 0)

	go GetAircraftsFromHttp(aircraft_data_channel, config, ticker_2hz)

	go ProcessAircraftData(aircraft_data_channel, config, midnight_ticker)

	if config.Upload_folder_path != "" {
		go UploadFilesToSharedFolder(config, three_am_ticker)
	} else {
		logger.Println("main: 'upload_folder_path' not specified in config yaml file.",
			"Saving the data locally.")
	}

	logger.Println("main: Started the radarcape listener.")
	logger.Println(
		"main: Listening on hostname", config.Radarcape_hostname,
		"for the following aircrafts:", config.Icao_aircraft_types,
	)

	waitForCloseInterrupt()
	midnight_ticker.Stop()

}
