package main

import (
	"fmt"
	"os"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	secondsInTheDay  = 24 * 3600
	minSecondsToLive = secondsInTheDay
	maxSecondsToLive = secondsInTheDay
)

func msgDialog(waitGroup *sync.WaitGroup, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	var msg tgbotapi.MessageConfig

	defer waitGroup.Done()

	/// check delete timeout and protect
	okAutoDelete, err := checkChatAutoDeleteTimer(bot, update)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] check auto delete: %s", err)
	}

	switch okAutoDelete {
	case true:
		/// check session
		/// check previos dialog
		/// overvise greeting
		msg = tgbotapi.NewMessage(update.Message.Chat.ID, MsgWelcome)
		msg.ReplyMarkup = wannabeKeyboard
		msg.ParseMode = tgbotapi.ModeMarkdown
	default:
		msg = tgbotapi.NewMessage(update.Message.Chat.ID, FatalUnwellSecurity)
		msg.ParseMode = tgbotapi.ModeMarkdown
	}

	if err := removeMsg(bot, update); err != nil {
		fmt.Fprintf(os.Stderr, "[!] remove: %s", err)
	}

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

func removeMsg(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	remove := tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
	if _, err := bot.Request(remove); err != nil {

		return fmt.Errorf("request: %w", err)
	}

	return nil
}

func checkChatAutoDeleteTimer(bot *tgbotapi.BotAPI, update tgbotapi.Update) (bool, error) {
	chat, err := bot.GetChat(
		tgbotapi.ChatInfoConfig{
			ChatConfig: tgbotapi.ChatConfig{
				ChatID: update.Message.Chat.ID},
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] get chat: %s", err)

		return false, fmt.Errorf("get chat: %w", err)
	}

	if chat.MessageAutoDeleteTime < minSecondsToLive || chat.MessageAutoDeleteTime > maxSecondsToLive {

		return false, nil
	}

	return true, nil
}
