package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
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

// Session - session.
type Session struct {
	OurMsgID int `json:"our_message_id"`
	Stage    int `json:"stage"`
}

func msgDialog(waitGroup *sync.WaitGroup, dbase *badger.DB, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	defer waitGroup.Done()

	defer func() {
		if err := removeMsg(bot, update.Message.Chat.ID, update.Message.MessageID); err != nil {
			fmt.Fprintf(os.Stderr, "[!] remove: %s", err)
		}
	}()

	/// check delete timeout and protect
	okAutoDelete, err := checkChatAutoDeleteTimer(bot, update)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] check auto delete: %s", err)

		return
	}

	if !okAutoDelete {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, FatalUnwellSecurity)
		msg.ParseMode = tgbotapi.ModeMarkdown

		if _, err := bot.Send(msg); err != nil {
			fmt.Fprintf(os.Stderr, "[!] send: %s", err)
		}

		return
	}

	/// check session
	var (
		data       []byte
		session    *Session = &Session{}
		int64bytes [8]byte
	)
	binary.BigEndian.PutUint64(int64bytes[:], uint64(update.Message.Chat.ID))
	key := sha256.Sum256(int64bytes[:])
	err = dbase.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key[:])
		if err != nil {
			return fmt.Errorf("db get: %w", err)
		}

		err = item.Value(func(v []byte) error {
			data = append([]byte{}, v...)

			return nil
		})
		if err != nil || err != badger.ErrKeyNotFound {
			return fmt.Errorf("db get: %w", err)
		}

		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] session: %s", err)

		return
	}

	if data != nil {
		err := json.Unmarshal(data, session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[!] session parse: %s", err)

			return
		}
	}

	/// check previos dialog
	/// overvise greeting
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, MsgWelcome)
	msg.ReplyMarkup = wannabeKeyboard
	msg.ParseMode = tgbotapi.ModeMarkdown

	if _, err := bot.Send(msg); err != nil {
		fmt.Fprintf(os.Stderr, "[!] send: %s", err)

		return
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

func removeMsg(bot *tgbotapi.BotAPI, chatID int64, msgID int) error {
	remove := tgbotapi.NewDeleteMessage(chatID, msgID)
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
