package main

import (
	"crypto/sha256"
	"log"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// DefaultUpdateTimeout - default messages updates timeout.
	DefaultUpdateTimeout = 3
	// DefaultSalt - salt.
	DefaultSalt = "we4;6prSfm_k+Gn"
	// DefaultIterations - KDF iters.
	DefaultIterations = 4096
	// DefaultKeyLen - default key len.
	DefaultKeyLen = 32
)

// Config - config.
type Config struct {
	Token      string
	UpdateTout int
	DebugLevel int
	Debug      bool
	DBDir      string
	DBKey      []byte
}

// configFromEnv - fill config from environment vars.
func configFromEnv() Config {
	var (
		debug bool
		key   []byte
	)

	token := os.Getenv("EMBASSY_TOKEN")               // Telegram token
	updateTout := os.Getenv("EMBASSY_UPDATE_TIMEOUT") // DefaultUpdateTimeout = 3
	debugLevel := os.Getenv("EMBASSY_DEBUG")          // number now, 0 is silent
	dbDir := os.Getenv("EMBASSY_BUDGER_DIR")          // Database dir, default db
	dbKey := os.Getenv("EMBASSY_BUDGER_KEY")

	if dbKey == "" {
		log.Panic("NO ENCRYPTION KEY")
	}

	tout, _ := strconv.Atoi(updateTout)
	if tout <= 0 {
		tout = DefaultUpdateTimeout
	}

	dbg, _ := strconv.Atoi(debugLevel)
	if dbg > 0 {
		debug = true
	}

	parts := strings.Split(dbKey, ":")
	switch len(parts) {
	case 1:
		key = pbkdf2.Key([]byte(dbKey), []byte(DefaultSalt), DefaultIterations, DefaultKeyLen, sha256.New)
	case 2:
		key = pbkdf2.Key([]byte(parts[1]), []byte(parts[0]), DefaultIterations, DefaultKeyLen, sha256.New)
	default:
		key = pbkdf2.Key([]byte(strings.TrimPrefix(dbKey, parts[0]+":")), []byte(parts[0]), DefaultIterations, DefaultKeyLen, sha256.New)
	}

	return Config{
		Token:      token,
		UpdateTout: tout,
		DebugLevel: dbg,
		Debug:      debug,
		DBDir:      dbDir,
		DBKey:      key,
	}
}
