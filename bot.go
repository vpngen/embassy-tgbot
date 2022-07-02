package main

import (
	"fmt"
	"os"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func createBot(token string, debug bool) (*tgbotapi.BotAPI, error) {

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	bot.Debug = debug

	if debug {
		fmt.Fprintf(os.Stderr, "[i] Authorized on account %s\n", bot.Self.UserName)
	}

	return bot, nil
}

func runBot(wg *sync.WaitGroup, stop <-chan struct{}, bot *tgbotapi.BotAPI, updateTout, debugLevel int) {
	defer wg.Done()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = updateTout

	updates := bot.GetUpdatesChan(u)

L1:
	for {
		select {
		case update := <-updates:
			if update.Message != nil { // If we got a message
				if bot.Debug {
					fmt.Fprintf(os.Stderr, "[%s] %s", update.Message.From.UserName, update.Message.Text)
				}

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
				msg.ReplyToMessageID = update.Message.MessageID

				bot.Send(msg)
			}
		case <-stop:
			fmt.Fprintln(os.Stdout, "[-] Run: Stop signal was received")
			break L1
		}
	}
}
