package main

import (
	"crypto/sha256"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/vpngen/embassy-tgbot/logs"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/ssh"
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

// DeptOpts - opts to ministry conn
type DeptOpts struct {
	sshConfig *ssh.ClientConfig
	controlIP string
	token     string
	fake      bool
}

// Config - config.
type Config struct {
	Token        string
	Token2       string
	UpdateTout   int
	DebugLevel   int
	BotDebug     bool
	DBDir        string
	DBKey        []byte
	SupportURL   string
	SupportEmail string
	ckChatID     int64
	Dept         DeptOpts
}

// configFromEnv - fill config from environment vars.
func configFromEnv() Config {
	var (
		debug bool
		key   []byte
	)

	token := os.Getenv("EMBASSY_TOKEN")               // Telegram token
	token2 := os.Getenv("CHECKBOT_TOKEN")             // Telegram token 2
	updateTout := os.Getenv("EMBASSY_UPDATE_TIMEOUT") // DefaultUpdateTimeout = 3
	debugLevel := os.Getenv("EMBASSY_DEBUG")          // number now, 0 is silent
	botDebug := os.Getenv("BOT_DEBUG")                // st now, is debug
	dbDir := os.Getenv("EMBASSY_BADGER_DIR")          // Database dir, default db
	dbKey := os.Getenv("EMBASSY_BADGER_KEY")
	supportURL := os.Getenv("SUPPORT_URL")
	supportEmail := os.Getenv("SUPPORT_EMAIL")
	ckChat := os.Getenv("CHECK_BILL_CHAT")
	ministryIP := os.Getenv("MINISTRY_IP")
	ministryToken := os.Getenv("MINISTRY_TOKEN")
	sshKeyPath := os.Getenv("SSHKEY_PATH")

	if dbKey == "" {
		log.Panic("NO ENCRYPTION KEY")
	}

	sshFake := false
	sshconf, err := createSSHConfig(sshKeyPath)
	if err != nil {
		if sshKeyPath != "FAKE" {
			log.Fatalf("NO SSH KEY: %s\n", err)
		}

		sshFake = true
	}

	if supportURL == "" {
		supportURL = DefaultSupportURL
	}

	tout, _ := strconv.Atoi(updateTout)
	if tout <= 0 {
		tout = DefaultUpdateTimeout
	}

	dbg, _ := strconv.Atoi(debugLevel)
	if dbg > int(logs.LevelDebug) {
		dbg = int(logs.LevelDebug)
	}

	if botDebug != "0" && botDebug != "" {
		debug = true
	}

	ckChatID, _ := strconv.ParseInt(ckChat, 10, 64)

	parts := strings.Split(dbKey, ":")
	switch len(parts) {
	case 1:
		key = pbkdf2.Key(
			[]byte(dbKey),
			[]byte(DefaultSalt),
			DefaultIterations,
			DefaultKeyLen,
			sha256.New,
		)
	case 2:
		key = pbkdf2.Key(
			[]byte(parts[1]),
			[]byte(parts[0]),
			DefaultIterations,
			DefaultKeyLen,
			sha256.New,
		)
	default:
		key = pbkdf2.Key(
			[]byte(strings.TrimPrefix(dbKey, parts[0]+":")),
			[]byte(parts[0]),
			DefaultIterations,
			DefaultKeyLen,
			sha256.New,
		)
	}

	return Config{
		Token:        token,
		Token2:       token2,
		UpdateTout:   tout,
		DebugLevel:   dbg,
		BotDebug:     debug,
		DBDir:        dbDir,
		DBKey:        key,
		SupportURL:   supportURL,
		SupportEmail: supportEmail,
		ckChatID:     ckChatID,
		Dept: DeptOpts{
			controlIP: ministryIP,
			sshConfig: sshconf,
			token:     ministryToken,
			fake:      sshFake,
		},
	}
}
