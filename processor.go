// Data Processing logic.
//
// Relevant functions which are needed to process and save the received data.
package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// Data processor goroutine.
//
// This Goroutine receives data from the receiver goroutine and saves the data to the corresponding
// output file. If the CsvGenerationLogic goroutine has generated a new set of CSV writers (due to
// date change), the old csv files are closed and we continue to write into the new csvs.
func ProcessAircraftData(aircraft_data_chan <-chan AircraftData, config Config, ticker *MidnightTicker) {
	aircraft_occurence_map := make(map[string]int)

	csv_writers_chan := make(chan map[string]CsvWriteCloser)

	go CsvGenerationLogic(csv_writers_chan, config, ticker)

	csv_writers := <-csv_writers_chan

	logger.Println("ProcessAircraftData: Started worker goroutine")

	for {
		select {
		case data := <-aircraft_data_chan:
			aircraft_occurence_map[data.Typ]++
			fmt.Print(aircraft_occurence_map, "\r")

			// Write the received data to the relevant csv
			err := csv_writers[data.Typ].Write(data.GetDataAsList())
			if err != nil {
				logger.Fatal(err)
			}

			// Flush the buffer
			csv_writers[data.Typ].Flush()

		case new_csv_writers := <-csv_writers_chan:
			for _, writer := range csv_writers {
				writer.Close()
			}
			csv_writers = new_csv_writers
			logger.Println("ProcessAircraftData: Changed csv writers in processAircaftData goroutine.")
			// Reset the occurence map for the new day
			aircraft_occurence_map = make(map[string]int)
		}

	}
}

// Highlevel CSV generation logic goroutine.
//
// This goroutine implements the logic with which new folders and csv files need to be
// generated. We first set a timer which block the execution of the goroutine until next
// midnight. Then, we enter the "normal" operating mode where we instantiate a ticker which
// ticks over every 24 hours, triggering the generation of new csv files.
func CsvGenerationLogic(csv_writers_chan chan map[string]CsvWriteCloser, config Config, ticker *MidnightTicker) {

	csv_writers_chan <- GenerateCsvWriters(time.Now(), config.Icao_aircraft_types)

	// Every time the ticker fires, we generate a new batch of csv files
	for ticker_time := range ticker.Processor_tick_chan {
		csv_writers_chan <- GenerateCsvWriters(time.Now(), config.Icao_aircraft_types)
		if DEBUG {
			logger.Println("CsvGenerationLogic: ticker rolled over:", ticker_time)
		}
	}
}

// CSV generator function.
//
// This method generates the folder path and the csv files where the data is saved. If
// a csv is already present for the given day we simply append to said csv, otherwise we generate
// a new one with the relevant header. We wrap the csv writer and the file in a `CsvWriteCloser` struct
// in order to close the file properly after writing to it.
func GenerateCsvWriters(date time.Time, aircrafts []string) map[string]CsvWriteCloser {
	logger.Println("GenerateCsvWriters: Generating CSV files")

	csv_writers := make(map[string]CsvWriteCloser, len(aircrafts))

	folder_path := getAppBasePath() + "Data/" + date.Format(dateFormatString) + "/"

	err := os.MkdirAll(folder_path, os.ModePerm)
	// If we get an error apart from `Folder already Exists` we break the execution.
	if !errors.Is(err, fs.ErrExist) && err != nil {
		logger.Fatal(err)
	}

	for _, aircraft_type := range aircrafts {
		file_path := folder_path + "output_file_" + aircraft_type + ".csv"
		file_does_not_exist := false

		// Check if file at the given file_path already exists.
		if _, err := os.Stat(file_path); errors.Is(err, os.ErrNotExist) {
			file_does_not_exist = true
		} else if err != nil {
			logger.Fatal(err)
		}

		// Open/Create the CSV file.
		csv_file, err := os.OpenFile(file_path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModePerm)
		if err != nil {
			logger.Fatal(err)
		}

		// Add the csv writer to the writers map.
		csv_writers[aircraft_type] = CsvWriteCloser{csv.NewWriter(csv_file), csv_file}

		// If the file was newly created we add the necessary header ot the csv file
		if file_does_not_exist {
			err = csv_writers[aircraft_type].Write(AircraftData{}.GetHeadersAsList())
			if err != nil {
				logger.Fatal(err)
			}
			csv_writers[aircraft_type].Flush()
		}
	}

	logger.Println("GenerateCsvWriters: CSV files generated.")
	return csv_writers
}
