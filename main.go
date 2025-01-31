package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	badger "github.com/dgraph-io/badger/v4"

	"github.com/vpngen/embassy-tgbot/logs"
)

const (
	dataKeyRotationDuration = 10 * 24 * time.Hour // 10 days
	defaultIndexCacheSize   = 100 << 20           // 100 Mb
)

func main() {
	cfg := configFromEnv()

	SetSupportMessages(cfg.SupportURL) // i dont know howto do this more clearely

	// set logs
	logs.SetLogLevel(int32(cfg.DebugLevel))

	// create a bot
	bot, err := createBot(cfg.Token)
	if err != nil {
		log.Panic(err)
	}

	// create a bot2
	bot2, err := createBot2(cfg.Token2)
	if err != nil {
		log.Panic(err)
	}

	dbopts := badger.DefaultOptions(cfg.DBDir).
		WithIndexCacheSize(defaultIndexCacheSize).
		WithEncryptionKey(cfg.DBKey).
		WithEncryptionKeyRotationDuration(dataKeyRotationDuration) // 10 days

	dbase, err := badger.Open(dbopts)
	if err != nil {
		log.Fatal(err)
	}

	defer dbase.Close()

	waitGroup := &sync.WaitGroup{}
	stop := make(chan struct{})

	// run the badger gc
	waitGroup.Add(1)

	go badgerGC(waitGroup, stop, dbase)

	// run the maintenance check
	if cfg.Maintenance != nil {
		fmt.Fprintf(os.Stderr, "[i] Maintenance check is enabled\n")

		waitGroup.Add(1)

		go checkMantenance(waitGroup, stop, bot2, cfg.ckChatID, cfg.Maintenance)
	}
	// run the bot
	waitGroup.Add(1)

	go runBot(waitGroup, stop, dbase, bot, cfg.UpdateTout, cfg.DebugLevel, cfg.Ministry, cfg.Maintenance, cfg.LabelStorage, cfg.sessionSecret, cfg.queueSecret)

	// run the bot2
	waitGroup.Add(1)

	go runBot2(waitGroup, stop, dbase, bot2, cfg.UpdateTout, cfg.DebugLevel, cfg.Maintenance)

	// run the QRun(2)
	waitGroup.Add(2)

	go ReceiptQueueLoop(waitGroup, dbase, stop, bot, bot2, cfg.ckChatID, cfg.Ministry, cfg.sessionSecret, cfg.queue2Secret, cfg.Maintenance)
	go ReceiptQueueLoop2(waitGroup, dbase, stop, bot, bot2, cfg.ckChatID)

	// run the stat sync
	waitGroup.Add(1)

	go statSyncLoop(waitGroup, stop, cfg.LabelStorage, cfg.Ministry)

	// catch exit signals
	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	for range kill {
		logs.Criticln("[-] Main: Stop signal was received")
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
	logs.Criticln("[-] Main routine was finished")
}

func badgerGC(wg *sync.WaitGroup, stop <-chan struct{}, db *badger.DB) {
	defer wg.Done()

	timer := time.NewTimer(5 * time.Minute)

	defer timer.Stop()

	for {
		select {
		case <-timer.C:
		again:
			err := db.RunValueLogGC(0.5)
			if err == nil {
				goto again
			}

			timer.Reset(5 * time.Minute)
		case <-stop:
			return
		}
	}
}
