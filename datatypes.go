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
	Icao_aircraft_types []string
	Radarcape_hostname  string
}

// Load configuration file from disk.
//
// The Config file is in the .yaml file format and contains info about which aircrafts are relevant to the study.
func (config *Config) LoadConfiguration(file string) {

	// config_ptr := new(Config)

	config_file, err := os.Open(file)

	if err != nil {
		logger.Fatal(err)
	}

	defer config_file.Close()

	err = yaml.NewDecoder(config_file).Decode(config)

	if err != nil {
		logger.Fatal(err)
	}

	if DEBUG {
		logger.Println("GetConfiguration: config:")
		logger.Println("GetConfiguration: ", *config)
	}

}

// This struct implements the structure of the received json data and stores the information of
// one aircraft at a certain time.
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

// Get the name of the fields of the AircraftData struct as a slice of strings.
func (ac_data AircraftData) GetHeadersAsList() []string {
	// Based on https://stackoverflow.com/a/67608475
	t := reflect.TypeOf(ac_data)

	aircraft_data_struct_field_names := make([]string, t.NumField())
	for i := range aircraft_data_struct_field_names {
		aircraft_data_struct_field_names[i] = t.Field(i).Name
	}
	return aircraft_data_struct_field_names
}

// Get the values of the fields of the AircraftData struct as a slice of strings.
func (ac_data AircraftData) GetDataAsList() []string {
	// Based on https://stackoverflow.com/a/67608475
	v := reflect.ValueOf(ac_data)

	aircraft_data_struct_field_names := make([]string, v.NumField())
	for i := range aircraft_data_struct_field_names {
		aircraft_data_struct_field_names[i] = fmt.Sprint(v.Field(i))
	}
	return aircraft_data_struct_field_names
}

// HttpGetterConfig wraps the config parameters for the receiver goroutine.
type HttpGetterConfig struct {
	Module_config_ptr *Config
	Ticker            *time.Ticker
}

// Wrapper struct to ensure files are properly closed once the csv.Writer is not needed anymore.
type CsvWriteCloser struct {
	*csv.Writer
	io.Closer
}

// Wrapper for the timer and tickers used for synchronisation of the goroutines.
//
// NOTE(@naefjo): If you do not use all the provided timers, the unused one will block
// the logi in the background and the other tickers will not be incremented anymore.
type MidnightTicker struct {
	Processor_tick_chan <-chan time.Time
	Uploader_tick_chan  <-chan time.Time
	halt                chan<- struct{} // singal channel to indicate that all the
	// channels should be closed.
}

// Wrapper function which instantiates and initializes the MidnightTicker struct.
//
// Inside the function we spin up a goroutine which runs in the background and sends
// time stamps from a timer and a ticker to the MidnightTicker channels.
// This allows us to conveniently use the provided channels without having to worry
// about the logic which is needed to fire the tickers at the right time.
func NewMidnightTicker() *MidnightTicker {

	mn_ticker_chan1 := make(chan time.Time, 1)
	mn_ticker_chan2 := make(chan time.Time, 1)
	mn_halt_chan := make(chan struct{})

	mn_ticker := &MidnightTicker{
		Processor_tick_chan: mn_ticker_chan1,
		Uploader_tick_chan:  mn_ticker_chan2,
		halt:                mn_halt_chan,
	}

	// Spin up goroutine which makes sure that the midnight tickers roll over at midnight.
	go func() {
		// Set up a timer which expires at midnight.
		curr_time := time.Now()
		midnight_timer := time.NewTimer(
			time.Until(
				time.Date(
					curr_time.Year(),
					curr_time.Month(),
					curr_time.Day()+1,
					0, 0, 10, 0,
					curr_time.Location(),
				),
			),
		)
		// midnight_timer := time.NewTimer(10 * time.Second)
		defer midnight_timer.Stop()

		// Blocking function which waits until either the timer expires or a
		// halt signal is sent to the MidnightTicker.
		err := func() error {
			select {
			case timer_time := <-midnight_timer.C:
				mn_ticker_chan1 <- timer_time
				mn_ticker_chan2 <- timer_time
				return nil

			case <-mn_halt_chan:
				// midnight_timer does not need to be stopped since we return
				// from the goroutine and the stop has been deferred.
				close(mn_ticker_chan1)
				close(mn_ticker_chan2)
				return errors.New("ticker has been halted early")
			}
		}()
		if err != nil {
			return
		}

		if DEBUG {
			logger.Println("init: timer rolled over.")
		}

		// Set up a ticker which triggers every 24 hours.
		// ticker_24hrs := time.NewTicker(10 * time.Second)
		ticker_24hrs := time.NewTicker(24 * time.Hour)
		defer ticker_24hrs.Stop()

		// Every time the ticker fires, we send the item to the MidnightTicker channels.
		for {
			select {
			case ticker_time := <-ticker_24hrs.C:
				mn_ticker_chan1 <- ticker_time
				mn_ticker_chan2 <- ticker_time

				if DEBUG {
					logger.Println("init: ticker rolled over.")
				}

			case <-mn_halt_chan:
				// ticker_24hrs does not need to be stopped since we return from the
				// goroutine and the stop has been deferred.
				close(mn_ticker_chan1)
				close(mn_ticker_chan2)
				return
			}
		}
	}()

	return mn_ticker
}

// Close the channels of the ticker.
//
// Send a signal to the goroutine which is spun up in the NewMidnightTicker
// function which stops the underlying timer/ticker and closes the ticker
// channels of the MidnightTicker struct.
func (ticker *MidnightTicker) Stop() {
	ticker.halt <- struct{}{}
	defer close(ticker.halt)
}
