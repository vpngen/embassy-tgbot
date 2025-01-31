package main

import (
	"fmt"
	"sync"

	"github.com/dgraph-io/badger/v4"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/vpngen/embassy-tgbot/logs"
)

func createBot(token string) (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	// bot.Debug = debug

	logs.Criticf("[i] Authorized on account %s\n", bot.Self.UserName)

	return bot, nil
}

func runBot(
	waitGroup *sync.WaitGroup,
	stop <-chan struct{},
	dbase *badger.DB,
	bot *tgbotapi.BotAPI,
	updateTout,
	debugLevel int,
	ministry MinistryOpts,
	maintenance *Maintenance,
	labelStorage *LabelStorage,
	sessionSecret []byte,
	queueSecret []byte,
) {
	defer waitGroup.Done()

	opts := handlerOpts{
		wg:    waitGroup,
		db:    dbase,
		bot:   bot,
		cw:    NewChatsWins(),
		debug: debugLevel,
		ls:    labelStorage,
		mnt:   maintenance,

		sessionSecret: sessionSecret,
		queueSecret:   queueSecret,
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = updateTout

	updates := bot.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			switch {
			case update.Message != nil: // If we got a message
				if update.Message.Chat.Type == "private" {
					logs.Debugf("[i] User: %s Message: %s\n", update.Message.From.UserName, update.Message.Text)

					waitGroup.Add(1)

					go messageHandler(opts, update, ministry)

					break
				}

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, InfoGroupsNotAllowedMessage)
				msg.ReplyToMessageID = update.Message.MessageID

				if _, err := bot.Send(msg); err != nil {
					logs.Debugf("[!] send: %s\n", err)
				}
			case update.CallbackQuery != nil:
				// Respond to the callback query, telling Telegram to show the user
				// a message with the data received.
				callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
				if _, err := bot.Request(callback); err != nil {
					logs.Debugf("[!] callback: %s\n", err)
				}

				waitGroup.Add(1)

				go buttonHandler(opts, update)
			}
		case <-stop:
			logs.Infoln("[-] Run: Stop signal was received")

			return
		}
	}
}
