// Data aqcuisition logic.
//
// Relevant functions and goroutines which are necessary to acquire the data from
// the Radarcape.

package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// Radarcape data getter goroutine.
//
// With a frequency specified by the ticker we first issue a GET request to the web server on the radarcape which
// yields us a json file with the current list of all the observable aircrafts and their respective infos.
// We then decode the json into a slice of AircraftData structs and subsequently filter out all the messages
// which are either not of interest to us or are duplicates. The remaining messages are then posted into a
// channel which sends them to a worker goroutine
func GetAircraftsFromHttp(aircraft_data_channel chan<- AircraftData,
	config Config, ticker *time.Ticker,
) {

	http_client := &http.Client{}

	connection_established := false

	// Hash map (i.e. Dict) where we store the most up to date message of each ICAO address.
	last_received_messages := make(map[string]AircraftData)

	aircraftlist_url := "http://" + config.Radarcape_hostname + "/aircraftlist.json"

	LogInfo("GetAircraftsFromHttp: Successfully started receiver goroutine.")

	for range ticker.C { // Block until new ticker update is received

		// Query the radarcape for a new json containing aircraft data.
		aircraft_list, err := RequestAircrafList(http_client, aircraftlist_url)

		// Check if the reported error is due to a read timeout.
		if err != nil {
			LogWarn(err)
			connection_established = false
			// Prevent spam on stdout.
			time.Sleep(20 * time.Second)
			continue
		} else if !connection_established {
			connection_established = true
			SignalConnectionEstablished()
		}

		// Send aircraft data to the processor goroutine.
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
func RequestAircrafList(http_client *http.Client, aircraftlist_url string) (aircraft_list []AircraftData, err error) {
	resp, err := http_client.Get(aircraftlist_url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&aircraft_list); err != nil {
		return nil, err
	}

	return
}

// Signals that a connection to the radarcape has been established.
//
// Currently only a log message on stdout. Could be extended to send a startup log to polybox
// or whatever.
func SignalConnectionEstablished() {
	LogInfo("SignalConnectionEstablished: Established a connection to the radarcape.")
}
