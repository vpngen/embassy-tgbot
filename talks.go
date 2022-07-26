package main

import (
	"fmt"
	"math/rand"
	"os"
	"sync"

	"github.com/dgraph-io/badger/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	secondsInTheDay  = 24 * 3600
	minSecondsToLive = secondsInTheDay
	maxSecondsToLive = secondsInTheDay
)

const (
	stageStart int = iota
	stageWait4Choice
	stageWait4Bill
	stageWait4Decision
	stageCleanup
)

func messageHandler(waitGroup *sync.WaitGroup, dbase *badger.DB, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	defer waitGroup.Done()

	ecode := fmt.Sprintf("%04x", rand.Int31()) // unique e-code

	defer func() {
		if err := removeMsg(bot, update.Message.Chat.ID, update.Message.MessageID); err != nil {
			fmt.Fprintf(os.Stderr, "[!:%s] remove: %s\n", ecode, err)
			// we don't want to handle this
		}
	}()

	/// check delete timeout and protect
	okAutoDelete, err := checkChatAutoDeleteTimer(bot, update.Message.Chat.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!:%s] check auto delete: %s\n", ecode, err)
		somethingWrong(bot, update.Message.Chat.ID, ecode)

		return
	}

	if !okAutoDelete {
		return
	}

	/// check session
	session, err := checkSession(dbase, update.Message.Chat.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!:%s] check session: %s\n", ecode, err)
		somethingWrong(bot, update.Message.Chat.ID, ecode)

		return
	}

	switch session.Stage {
	case stageWait4Choice, stageWait4Decision, stageCleanup:
		return

	case stageWait4Bill:
		err := sendAttestationAssignedMessage(bot, dbase, update.Message.Chat.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[!:%s] bill recv: %s\n", ecode, err)
			somethingWrong(bot, update.Message.Chat.ID, ecode)
		}

		defer func() {
			if err := removeMsg(bot, update.Message.Chat.ID, session.OurMsgID); err != nil {
				fmt.Fprintf(os.Stderr, "[!:%s] remove old: %s\n", ecode, err)
				// we don't want to handle this
			}
		}()

	default:
		err := sendWelcomeMessage(bot, dbase, update.Message.Chat.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[!:%s] welcome: %s\n", ecode, err)
			somethingWrong(bot, update.Message.Chat.ID, ecode)
		}
	}
}

func buttonHandler(waitGroup *sync.WaitGroup, dbase *badger.DB, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	defer waitGroup.Done()

	ecode := fmt.Sprintf("%04x", rand.Int31()) // unique error code

	/// check delete timeout and protect
	okAutoDelete, err := checkChatAutoDeleteTimer(bot, update.CallbackQuery.Message.Chat.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!:%s] check auto delete: %s\n", ecode, err)
		somethingWrong(bot, update.CallbackQuery.Message.Chat.ID, ecode)

		return
	}

	if !okAutoDelete {
		return
	}

	/// check session
	session, err := checkSession(dbase, update.CallbackQuery.Message.Chat.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!:%s] check session: %s\n", ecode, err)
		somethingWrong(bot, update.CallbackQuery.Message.Chat.ID, ecode)

		return
	}

	switch {
	case update.CallbackQuery.Data == "wannabe" && session.Stage == stageWait4Choice:
		if err := sendQuizMessage(bot, dbase, update.CallbackQuery.Message.Chat.ID); err != nil {
			fmt.Fprintf(os.Stderr, "[!:%s] wannabe: %s\n", ecode, err)
			somethingWrong(bot, update.CallbackQuery.Message.Chat.ID, ecode)
		}

		defer func() {
			if err := removeMsg(bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
				fmt.Fprintf(os.Stderr, "[!:%s] remove: %s\n", ecode, err)
				// we don't want to handle this
			}
		}()
	}
}

func removeMsg(bot *tgbotapi.BotAPI, chatID int64, msgID int) error {
	remove := tgbotapi.NewDeleteMessage(chatID, msgID)
	if _, err := bot.Request(remove); err != nil {

		return fmt.Errorf("request: %w", err)
	}

	return nil
}

func somethingWrong(bot *tgbotapi.BotAPI, chatID int64, ecode string) {
	text := fmt.Sprintf("%s: код %s", FatalSomeThingWrong, ecode)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ProtectContent = true

	if _, err := bot.Send(msg); err != nil {
		fmt.Fprintf(os.Stderr, "[!:%s] SOMETHING WRONG: %s\n", ecode, err)
	}
}

func sendWelcomeMessage(bot *tgbotapi.BotAPI, dbase *badger.DB, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, MsgWelcome)
	msg.ReplyMarkup = wannabeKeyboard
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ProtectContent = true

	newMsg, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(dbase, newMsg.Chat.ID, newMsg.MessageID, stageWait4Choice)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

func sendQuizMessage(bot *tgbotapi.BotAPI, dbase *badger.DB, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, MsgQuiz)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ProtectContent = true

	newMsg, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(dbase, newMsg.Chat.ID, newMsg.MessageID, stageWait4Bill)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

func sendAttestationAssignedMessage(bot *tgbotapi.BotAPI, dbase *badger.DB, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, MsgAttestationAssigned)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ProtectContent = true

	newMsg, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(dbase, newMsg.Chat.ID, newMsg.MessageID, stageWait4Decision)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

func checkChatAutoDeleteTimer(bot *tgbotapi.BotAPI, chatID int64) (bool, error) {
	chat, err := bot.GetChat(
		tgbotapi.ChatInfoConfig{
			ChatConfig: tgbotapi.ChatConfig{
				ChatID: chatID},
		},
	)
	if err != nil {
		return false, fmt.Errorf("get chat: %w", err)
	}

	if chat.MessageAutoDeleteTime < minSecondsToLive || chat.MessageAutoDeleteTime > maxSecondsToLive {
		msg := tgbotapi.NewMessage(chatID, FatalUnwellSecurity)
		msg.ParseMode = tgbotapi.ModeMarkdown

		if _, err := bot.Send(msg); err != nil {
			return false, fmt.Errorf("send: %w", err)
		}

		return false, nil
	}

	return true, nil
}
