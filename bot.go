package main

import (
	"fmt"
	"os"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	wannabeKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Хочу свой VPN", "wannabe"),
		),
	)
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
				if update.Message.Chat.Type == "private" {
					if bot.Debug {
						fmt.Fprintf(os.Stderr, "[%s] %s", update.Message.From.UserName, update.Message.Text)
					}

					// !!! implement handling return vars
					//					sendReply(bot, update, update.Message.Text)
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, MsgWelcome)
					msg.ReplyToMessageID = update.Message.MessageID
					msg.ReplyMarkup = wannabeKeyboard

					bot.Send(msg)

					break
				}

				sendReply(bot, update, WarnGroupsNotAllowed)
			}
		case <-stop:
			fmt.Fprintln(os.Stdout, "[-] Run: Stop signal was received")

			bot.StopReceivingUpdates()

			break L1
		}
	}
}

// sendReply - send text
func sendReply(bot *tgbotapi.BotAPI, update tgbotapi.Update, text string) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ReplyToMessageID = update.Message.MessageID

	message, err := bot.Send(msg)

	return message, fmt.Errorf("send msg: %w", err)
}
