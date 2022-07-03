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
			switch {
			case update.Message != nil: // If we got a message
				if update.Message.Chat.Type == "private" {
					if bot.Debug {
						fmt.Fprintf(os.Stderr, "[i] User: %s Message: %s", update.Message.From.UserName, update.Message.Text)
					}

					// !!! implement handling return vars
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, MsgWelcome)
					msg.ReplyToMessageID = update.Message.MessageID
					msg.ReplyMarkup = wannabeKeyboard

					if _, err := bot.Send(msg); err != nil {
						fmt.Fprintf(os.Stderr, "[!] send: %s", err)
					}

					break
				}

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, WarnGroupsNotAllowed)
				msg.ReplyToMessageID = update.Message.MessageID

				if _, err := bot.Send(msg); err != nil {
					fmt.Fprintf(os.Stderr, "[!] send: %s", err)
				}
			case update.CallbackQuery != nil:
				// Respond to the callback query, telling Telegram to show the user
				// a message with the data received.
				callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
				if _, err := bot.Request(callback); err != nil {
					fmt.Fprintf(os.Stderr, "[!] callback: %s", err)
				}

				if update.CallbackQuery.Data == "wannabe" {
					// And finally, send a message containing the data received.
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, MsgQuiz)
					if _, err := bot.Send(msg); err != nil {
						fmt.Fprintf(os.Stderr, "[!] send: %s", err)
					}
				}
			}
		case <-stop:
			fmt.Fprintln(os.Stdout, "[-] Run: Stop signal was received")

			break L1
		}
	}
}
