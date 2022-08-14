package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/vpngen/embassy-tgbot/logs"
)

const (
	dataKeyRotationDuration = 10 * 24 * time.Hour // 10 days
	defaultIndexCacheSize   = 100 << 20           // 100 Mb
)

func main() {
	cfg := configFromEnv()

	SetWannaKeyboard(cfg.SupportURL) // i dont know howto do this more clearely

	// set logs
	logs.SetLogLevel(int32(cfg.DebugLevel))

	// create a bot
	bot, err := createBot(cfg.Token, cfg.BotDebug)
	if err != nil {
		log.Panic(err)
	}

	// create a bot2
	bot2, err := createBot2(cfg.Token2, cfg.BotDebug)
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

	// run the bot
	waitGroup.Add(1)

	go runBot(waitGroup, stop, dbase, bot, cfg.UpdateTout, cfg.DebugLevel)

	// run the bot2
	waitGroup.Add(1)

	go runBot2(waitGroup, stop, dbase, bot2, cfg.UpdateTout, cfg.DebugLevel)

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

	select {
	case <-timer.C:
	again:
		err := db.RunValueLogGC(0.5)
		if err == nil {
			goto again
		}
	case <-stop:
		return
	}
}
