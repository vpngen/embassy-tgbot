package main

import (
	"fmt"
	"sync"

	"github.com/dgraph-io/badger/v4"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/vpngen/embassy-tgbot/logs"
)

func createBot2(token string, debug bool) (*tgbotapi.BotAPI, error) {
	bot2, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("create bot2: %w", err)
	}

	// bot.Debug = debug

	logs.Criticf("[i] Authorized on account %s\n", bot2.Self.UserName)

	return bot2, nil
}

func runBot2(
	waitGroup *sync.WaitGroup,
	stop <-chan struct{},
	dbase *badger.DB,
	bot2 *tgbotapi.BotAPI,
	updateTout,
	debugLevel int,
	maintenanceModeFull string,
	maintenanceModeNew string,
) {
	defer waitGroup.Done()

	opts := handlerOpts{
		wg:    waitGroup,
		db:    dbase,
		bot:   bot2,
		debug: debugLevel,
		mmf:   maintenanceModeFull,
		mmn:   maintenanceModeNew,
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = updateTout

	updates := bot2.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			switch {
			case update.Message != nil: // If we got a message
				logs.Debugf("[i] User: %s ChatID: %d Message: %s\n", update.Message.From.UserName, update.Message.Chat.ID, update.Message.Text)

				if update.Message.Chat.Type == "private" {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, InfoPrivateNotAllowedMessage)
					msg.ReplyToMessageID = update.Message.MessageID

					if _, err := bot2.Send(msg); err != nil {
						logs.Debugf("[!] send: %s\n", err)
					}

					break
				}

				waitGroup.Add(1)

				go messageHandler2(opts, update)

			case update.CallbackQuery != nil:
				// Respond to the callback query, telling Telegram to show the user
				// a message with the data received.
				callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
				if _, err := bot2.Request(callback); err != nil {
					logs.Debugf("[!] callback: %s\n", err)
				}

				waitGroup.Add(1)

				go buttonHandler2(opts, update)
			}
		case <-stop:
			logs.Infoln("[-] Run: Stop signal was received")

			return
		}
	}
}
