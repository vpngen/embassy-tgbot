package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	badger "github.com/dgraph-io/badger/v3"
)

const (
	dataKeyRotationDuration = 10 * 24 * time.Hour // 10 days
	defaultIndexCacheSize   = 100 << 20           // 100 Mb
)

func main() {
	cfg := configFromEnv()

	// create a bot
	bot, err := createBot(cfg.Token, cfg.Debug)
	if err != nil {
		log.Panic(err)
	}

	dbopts := badger.DefaultOptions(cfg.DBDir).
		WithIndexCacheSize(defaultIndexCacheSize).
		WithEncryptionKey(cfg.DBKey).
		WithEncryptionKeyRotationDuration(dataKeyRotationDuration) // 10 days

	db, err := badger.Open(dbopts)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	waitGroup := &sync.WaitGroup{}
	stop := make(chan struct{})

	// run the bot
	waitGroup.Add(1)

	go runBot(waitGroup, stop, bot, cfg.UpdateTout, cfg.DebugLevel)

	// catch exit signals
	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	for range kill {
		fmt.Fprintln(os.Stdout, "[-] Main: Stop signal was received")
		// avoid message loosing
		bot.StopReceivingUpdates()
		time.Sleep(time.Second * time.Duration(cfg.UpdateTout))
		// stop!
		close(stop)
		// bye bye!
		break
	}

	// stop app
	waitGroup.Wait()
	fmt.Fprintln(os.Stdout, "[-] Main routine was finished")
}
