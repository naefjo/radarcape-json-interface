// Data aqcuisition logic.
//
// Relevant functions and goroutines which are necessary to acquire the data from
// the Radarcape.
package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// Radarcape data getter goroutine.
//
// With a frequency of 2Hz we first issue a GET request to the web server on the radarcape which
// yields us a json file with the current list of all the observable aircrafts and their respective infos.
// We then decode the json into a slice of AircraftData structs and subsequently filter out all the messages
// which are either not of interest to us or are duplicates. The remaining messages are then posted into a
// channel which sends them to a worker goroutine
func GetAircraftsFromHttp(aircraft_data_channel chan<- AircraftData,
	config Config, ticker *time.Ticker,
) {

	http_client := &http.Client{}

	last_received_messages := make(map[string]AircraftData)

	aircraftlist_url := "http://" + config.Radarcape_hostname + "/aircraftlist.json"

	logger.Println("GetAircraftsFromHttp: Started receiver goroutine.")

	for range ticker.C { // Block until new ticker update is received

		aircraft_list, err := HttpRequest(http_client, aircraftlist_url)

		// Check if the reported error is due to a read timeout.
		if err != nil {
			timeout_err, ok := err.(*url.Error)
			logger.Println(timeout_err, ok, timeout_err.Timeout(), timeout_err.Temporary())
		}
		if timeout_err, ok := err.(*url.Error); ok && timeout_err.Timeout() {
			logger.Println("GetAircraftsFromHttp: Read timeout. skipping this loop iteration.")
			continue
		} else if err != nil {
			// TODO(@naefjo): For the moment, just log the error and retry so we can catch all the errors
			// that could occur.
			logger.Println(err)
			continue
		}

		for _, aircraft := range aircraft_list {
			// Check whether aircraft type is of interest to us and if we already received this
			// identical message.
			if IsInSlice(aircraft.Typ, config.Icao_aircraft_types) &&
				aircraft != last_received_messages[aircraft.Typ] {
				// Update our last received message map and send the message to the worker goroutine.
				last_received_messages[aircraft.Typ] = aircraft
				aircraft_data_channel <- aircraft
			}
		}
	}

}

// Wrapper function for opening of the http request.
//
// Makes sure that resources are released properly.
func HttpRequest(http_client *http.Client, aircraftlist_url string) (aircraft_list []AircraftData, err error) {
	resp, err := http_client.Get(aircraftlist_url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&aircraft_list)
	if err != nil {
		return nil, err
	}

	return
}
