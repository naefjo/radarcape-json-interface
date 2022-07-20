// Data Processing logic.
//
// Relevant functions which are needed to process and save the received data.

package main

import (
	"encoding/csv"
	"errors"
	"os"
	"time"
)

// Data processor goroutine.
//
// This Goroutine receives data from the receiver goroutine and saves the data to the corresponding
// output file. If the CsvGenerationLogic goroutine has generated a new set of CSV writers (due to
// date change), the old csv files are closed and we continue to write into the new csvs.
func ProcessAircraftData(aircraft_data_chan <-chan AircraftData, config Config, ticker *TimeTicker) {
	csv_writers_chan := make(chan map[string]CsvWriteCloser)

	go CsvGenerationLogic(csv_writers_chan, config, ticker)

	csv_writers := <-csv_writers_chan

	LogInfo("ProcessAircraftData: Successfully started worker goroutine.")

	for {
		select {
		case data := <-aircraft_data_chan:

			// Write the received data to the relevant csv.
			if err := csv_writers[data.Typ].Write(data.GetDataAsList()); err != nil {
				LogError(err)
			}

			// Flush the buffer.
			csv_writers[data.Typ].Flush()

		case new_csv_writers := <-csv_writers_chan:

			for _, writer := range csv_writers {
				writer.Close()
			}
			csv_writers = new_csv_writers
			LogInfo("ProcessAircraftData: Changed csv writers in processAircaftData goroutine.")
		}

	}
}

// Highlevel CSV generation logic goroutine.
//
// Every time the provided ticker triggers, we change the CSV writers to a new date.
func CsvGenerationLogic(csv_writers_chan chan map[string]CsvWriteCloser, config Config, ticker *TimeTicker) {

	csv_writers_chan <- GenerateCsvWriters(time.Now(), config.Icao_aircraft_types)

	// Every time the ticker fires, we generate a new batch of csv files
	for ticker_time := range ticker.Processor_tick_chan {
		csv_writers_chan <- GenerateCsvWriters(time.Now(), config.Icao_aircraft_types)
		if DEBUG {
			LogInfo("CsvGenerationLogic: ticker rolled over:", ticker_time)
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
	if DEBUG {
		LogInfo("GenerateCsvWriters: Generating CSV files")
	}

	csv_writers := make(map[string]CsvWriteCloser, len(aircrafts))

	folder_path := getDataFolder(date)

	if err := createFolder(folder_path); err != nil {
		LogError(err)
	}

	for _, aircraft_type := range aircrafts {
		file_path := folder_path + "output_file_" + aircraft_type + ".csv"
		file_does_not_exist := false

		// Check if file at the given file_path already exists.
		if _, err := os.Stat(file_path); errors.Is(err, os.ErrNotExist) {
			file_does_not_exist = true
		} else if err != nil {
			LogError(err)
		}

		// Open/Create the CSV file.
		csv_file, err := os.OpenFile(file_path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModePerm)
		if err != nil {
			LogError(err)
		}

		// Add the csv writer to the writers map.
		csv_writers[aircraft_type] = CsvWriteCloser{csv.NewWriter(csv_file), csv_file}

		// If the file was newly created we add the necessary header ot the csv file.
		if file_does_not_exist {
			err = csv_writers[aircraft_type].Write(AircraftData{}.GetHeadersAsList())
			if err != nil {
				LogError(err)
			}
			csv_writers[aircraft_type].Flush()
		}
	}

	LogInfo("GenerateCsvWriters: New CSV files generated.")
	return csv_writers
}
