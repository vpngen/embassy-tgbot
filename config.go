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

// MinistryOpts - opts to ministry conn
type MinistryOpts struct {
	sshConfig *ssh.ClientConfig
	controlIP string
	token     string
	fake      bool
}

// Config - config.
type Config struct {
	Token         string
	Token2        string
	UpdateTout    int
	DebugLevel    int
	BotDebug      bool
	DBDir         string
	DBKey         []byte
	SupportURL    string
	ckChatID      int64
	Ministry      MinistryOpts
	Maintenance   *Maintenance
	LabelStorage  *LabelStorage
	sessionSecret []byte
	queueSecret   []byte
	queue2Secret  []byte
}

// configFromEnv - fill config from environment vars.
func configFromEnv() Config {
	var (
		debug bool
		ls    *LabelStorage
	)

	token := os.Getenv("EMBASSY_TOKEN")               // Telegram token
	token2 := os.Getenv("CHECKBOT_TOKEN")             // Telegram token 2
	updateTout := os.Getenv("EMBASSY_UPDATE_TIMEOUT") // DefaultUpdateTimeout = 3
	debugLevel := os.Getenv("EMBASSY_DEBUG")          // number now, 0 is silent
	botDebug := os.Getenv("BOT_DEBUG")                // st now, is debug
	dbDir := os.Getenv("EMBASSY_BADGER_DIR")          // Database dir, default db
	dbKey := os.Getenv("EMBASSY_BADGER_KEY")
	supportURL := os.Getenv("SUPPORT_URL")
	ckChat := os.Getenv("CHECK_BILL_CHAT")
	ministryIP := os.Getenv("MINISTRY_IP")
	ministryToken := os.Getenv("MINISTRY_TOKEN")
	sshKeyPath := os.Getenv("SSHKEY_PATH")
	labelFilename := os.Getenv("LABEL_FILENAME")
	maintenanceStateFilesDir := os.Getenv("MAINTENANCE_STATE_FILES_DIR")

	sessionSecret := os.Getenv("SESSION_SECRET")
	queueSecret := os.Getenv("QUEUE_SECRET")

	if sessionSecret == "" {
		log.Fatal("NO SESSION SECRET")
	}

	if queueSecret == "" {
		log.Fatal("NO QUEUE SECRET")
	}

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
		supportURL = DefaultSupportURLText
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

	ls, err = NewLabelStorage(labelFilename)
	if err != nil {
		log.Panic(err)
	}

	return Config{
		Token:      token,
		Token2:     token2,
		UpdateTout: tout,
		DebugLevel: dbg,
		BotDebug:   debug,
		DBDir:      dbDir,
		DBKey:      genKeyFromEnv(dbKey, DefaultIterations, DefaultKeyLen),
		SupportURL: supportURL,
		ckChatID:   ckChatID,
		Ministry: MinistryOpts{
			controlIP: ministryIP,
			sshConfig: sshconf,
			token:     ministryToken,
			fake:      sshFake,
		},
		LabelStorage: ls,

		Maintenance: NewMantenance(maintenanceStateFilesDir),

		sessionSecret: genKeyFromEnv(sessionSecret, DefaultIterations, DefaultKeyLen),
		queueSecret:   genKeyFromEnv(queueSecret, DefaultIterations, DefaultKeyLen),
	}
}

func genKeyFromEnv(key string, iter, sz int) []byte {
	parts := strings.Split(key, ":")
	switch len(parts) {
	case 1:
		return pbkdf2.Key(
			[]byte(key),
			[]byte(DefaultSalt),
			iter,
			sz,
			sha256.New,
		)
	case 2:
		return pbkdf2.Key(
			[]byte(parts[1]),
			[]byte(parts[0]),
			iter,
			sz,
			sha256.New,
		)
	default:
		return pbkdf2.Key(
			[]byte(strings.TrimPrefix(key, parts[0]+":")),
			[]byte(parts[0]),
			iter,
			sz,
			sha256.New,
		)
	}
}
