// Datatypes and their member methods which are used throughout this project.

package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"time"

	"gopkg.in/yaml.v2"
)

// Config struct implements and groups the config parameters of this module.
type Config struct {
	Icao_aircraft_types []string `yaml:"icao_aircraft_types"`
	Radarcape_hostname  string   `yaml:"radarcape_hostname"`
	Upload_folder_path  string   `yaml:"upload_folder_path"`
	Backup_folder_path  string   `yaml:"backup_folder_path"`
}

// Load configuration file from disk.
//
// The Config file is in the .yaml file format and contains info about
// which aircrafts are relevant to the study.
func (config *Config) LoadConfiguration(file string) {

	// config_ptr := new(Config)

	config_file, err := os.Open(file)

	if err != nil {
		LogError(err)
	}

	defer config_file.Close()

	err = yaml.NewDecoder(config_file).Decode(config)

	if err != nil {
		LogError(err)
	}

	if DEBUG {
		LogInfo("GetConfiguration: config:")
		LogInfo("GetConfiguration: ", *config)
	}

}

// This struct implements the structure of the received json data
// and stores the information of one aircraft at a certain time.
type AircraftData struct {
	Alr  int     `json:"alr"`
	Alt  int     `json:"alt"`
	Altg int     `json:"altg"`
	Alts int     `json:"alts"`
	Ape  bool    `json:"ape"`
	Ava  string  `json:"ava"`
	Cat  string  `json:"cat"`
	Cou  string  `json:"cou"`
	Dbm  int     `json:"dbm"`
	Dis  float32 `json:"dis"`
	Dst  string  `json:"dst"`
	Fli  string  `json:"fli"`
	Gda  string  `json:"gda"`
	Hex  string  `json:"hex"`
	Lat  float64 `json:"lat"`
	Lla  int     `json:"lla"`
	Lon  float64 `json:"lon"`
	Mop  int     `json:"mop"`
	Nacp int     `json:"nacp"`
	Ns   uint32  `json:"ns"`
	Opr  string  `json:"opr"`
	Org  string  `json:"org"`
	Pic  int     `json:"pic"`
	Qnhs float32 `json:"qnhs"`
	Reg  string  `json:"reg"`
	Sda  int     `json:"sda"`
	Sil  int     `json:"sil"`
	Spd  int     `json:"spd"`
	Spi  bool    `json:"spi"`
	Squ  string  `json:"squ"`
	Src  string  `json:"src"`
	Tcm  int     `json:"tcm"`
	Tmp  int     `json:"tmp"`
	Trk  int     `json:"trk"`
	Tru  int     `json:"tru"`
	Typ  string  `json:"typ"`
	Uti  uint64  `json:"uti"`
	Vrt  int     `json:"vrt"`
	Wdi  int     `json:"wdi"`
	Wsp  int     `json:"wsp"`
}

// Get the name of the fields of the AircraftData struct as a
// slice of strings.
func (ac_data AircraftData) GetHeadersAsList() []string {
	// Based on https://stackoverflow.com/a/67608475
	t := reflect.TypeOf(ac_data)

	aircraft_data_struct_field_names := make([]string, t.NumField())
	for i := range aircraft_data_struct_field_names {
		aircraft_data_struct_field_names[i] = t.Field(i).Name
	}
	return aircraft_data_struct_field_names
}

// Get the values of the fields of the AircraftData struct as a
// slice of strings.
func (ac_data AircraftData) GetDataAsList() []string {
	// Based on https://stackoverflow.com/a/67608475
	v := reflect.ValueOf(ac_data)

	aircraft_data_struct_field_names := make([]string, v.NumField())
	for i := range aircraft_data_struct_field_names {
		aircraft_data_struct_field_names[i] = fmt.Sprint(v.Field(i))
	}
	return aircraft_data_struct_field_names
}

// Wrapper struct to ensure files are properly closed once the csv.Writer
// is not needed anymore.
type CsvWriteCloser struct {
	*csv.Writer
	io.Closer
}

// Wrapper for the timer and tickers used for synchronisation of the goroutines.
type TimeTicker struct {
	Processor_tick_chan <-chan time.Time
	halt                chan<- struct{} // singal channel to indicate that all the
	// channels should be closed.
}

// Wrapper function which instantiates and initializes the TimeTicker struct.
//
// Inside the function we spin up a goroutine which runs in the background and sends
// time stamps from a timer and a ticker to the TimeTicker channels.
// This allows us to conveniently use the provided channels without having to worry
// about the logic which is needed to fire the tickers at the right time.
func NewTimeTicker(hour, minute, second int) *TimeTicker {

	time_ticker_chan1 := make(chan time.Time, 1)
	time_halt_chan := make(chan struct{})

	time_ticker := &TimeTicker{
		Processor_tick_chan: time_ticker_chan1,
		halt:                time_halt_chan,
	}

	// Spin up goroutine which makes sure that the
	// tickers roll over at the specified time.
	go func() {
		// Set up a timer which expires at specified time on the next day.
		curr_time := time.Now()

		// NOTE(@naefjo): time.Date normalizes dates (e.g. Oct. 32 == Nov. 1).
		time_timer := time.NewTimer(
			time.Until(
				time.Date(
					curr_time.Year(),
					curr_time.Month(),
					curr_time.Day()+1,
					hour, minute, second, 0,
					curr_time.Location(),
				),
			),
		)
		defer time_timer.Stop()

		// Blocking function which waits until either the timer
		// expires or a halt signal is sent to the MidnightTicker.
		err := func() error {
			select {
			case timer_time := <-time_timer.C:
				time_ticker_chan1 <- timer_time
				return nil

			case <-time_halt_chan:
				// time_timer does not need to be stopped since
				// we return from the goroutine and the stop has
				// been deferred.
				close(time_ticker_chan1)
				return errors.New("ticker has been halted early")
			}
		}()
		if err != nil {
			LogError(err)
		}

		if DEBUG {
			LogInfo("init: timer rolled over.")
		}

		// Set up a ticker which triggers every 24 hours.
		ticker_24hrs := time.NewTicker(24 * time.Hour)
		defer ticker_24hrs.Stop()

		// Every time the ticker fires, we send the item to the
		// TimeTicker channels.
		for {
			select {
			case ticker_time := <-ticker_24hrs.C:
				time_ticker_chan1 <- ticker_time

				if DEBUG {
					LogInfo("init: ticker rolled over.")
				}

			case <-time_halt_chan:
				// ticker_24hrs does not need to be stopped since
				// we return from the goroutine and the stop has
				// been deferred.
				close(time_ticker_chan1)
				return
			}
		}
	}()

	return time_ticker
}

// Close the channels of the ticker.
//
// Send a signal to the goroutine which is spun up in the NewTimeTicker
// function which stops the underlying timer/ticker and closes the ticker
// channels of the TimeTicker struct.
func (ticker *TimeTicker) Stop() {
	ticker.halt <- struct{}{}
	defer close(ticker.halt)
}
