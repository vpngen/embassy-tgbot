package main

import (
	"os"
	"strconv"
)

// DefaultUpdateTimeout - default messages updates timeout
const DefaultUpdateTimeout = 3

// Config - config
type Config struct {
	Token      string
	UpdateTout int
	DebugLevel int
	Debug      bool
}

// configFromEnv - fill config from environment vars
func configFromEnv() Config {
	var debug bool

	token := os.Getenv("EMBASSY_TOKEN")       // Telegram token
	updateTout := os.Getenv("UPDATE_TIMEOUT") // DefaultUpdateTimeout = 3
	debugLevel := os.Getenv("EMBASSY_DEBUG")  // number now, 0 is silent

	dt, _ := strconv.Atoi(updateTout)
	if dt <= 0 {
		dt = DefaultUpdateTimeout
	}

	dbg, _ := strconv.Atoi(debugLevel)
	if dbg > 0 {
		debug = true
	}

	return Config{
		Token:      token,
		UpdateTout: dt,
		DebugLevel: dbg,
		Debug:      debug,
	}
}
