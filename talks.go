package main

import (
	"fmt"
	"os"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func msgDialog(waitGroup *sync.WaitGroup, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	defer waitGroup.Done()

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, MsgWelcome)
	msg.ReplyToMessageID = update.Message.MessageID
	msg.ReplyMarkup = wannabeKeyboard
	msg.ParseMode = tgbotapi.ModeMarkdown

	if _, err := bot.Send(msg); err != nil {
		fmt.Fprintf(os.Stderr, "[!] send: %s", err)
	}
}

func buttonHandling(waitGroup *sync.WaitGroup, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	defer waitGroup.Done()

	if update.CallbackQuery.Data == "wannabe" {
		// And finally, send a message containing the data received.
		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, MsgQuiz)
		msg.ParseMode = tgbotapi.ModeMarkdown

		if _, err := bot.Send(msg); err != nil {
			fmt.Fprintf(os.Stderr, "[!] send: %s", err)
		}
	}
}
