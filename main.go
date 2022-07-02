package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {

	cfg := configFromEnv()

	// create a bot
	bot, err := createBot(cfg.Token, cfg.Debug)
	if err != nil {
		log.Panic(err)
	}

	wg := &sync.WaitGroup{}
	stop := make(chan struct{})

	// run the bot
	wg.Add(1)
	go runBot(wg, stop, bot, cfg.UpdateTout, cfg.DebugLevel)

	// catch exit signals
	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

MainLoop:
	for {
		select {
		case <-kill:
			fmt.Fprintln(os.Stdout, "[-] Main: Stop signal was received")
			close(stop)
			break MainLoop
		}
	}

	// stop app
	wg.Wait()
	fmt.Fprintln(os.Stdout, "[-] Main routine was finished")
}
