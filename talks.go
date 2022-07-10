package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

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
	OurMsgID   int   `json:"our_message_id"`
	Stage      int   `json:"stage"`
	UpdateTime int64 `json:"updatetime"`
}

func msgDialog(waitGroup *sync.WaitGroup, dbase *badger.DB, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	defer waitGroup.Done()

	ecode := fmt.Sprintf("%04x", rand.Int31()) // unique error code

	defer func() {
		if err := removeMsg(bot, update.Message.Chat.ID, update.Message.MessageID); err != nil {
			fmt.Fprintf(os.Stderr, "[!:%s] remove: %s\n", ecode, err)
			somethingWrong(bot, update.Message.Chat.ID, ecode)
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
	case stageZero:
		return
	case stageQuiz:
		err := sendAttestationAssignedMessage(bot, dbase, update.Message.Chat.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[!:%s] bill recv: %s\n", ecode, err)
			somethingWrong(bot, update.Message.Chat.ID, ecode)
		}

		defer func() {
			if err := removeMsg(bot, update.Message.Chat.ID, session.OurMsgID); err != nil {
				fmt.Fprintf(os.Stderr, "[!:%s] remove old: %s\n", ecode, err)
				somethingWrong(bot, update.Message.Chat.ID, ecode)
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

func buttonHandling(waitGroup *sync.WaitGroup, dbase *badger.DB, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	defer waitGroup.Done()

	ecode := fmt.Sprintf("%04x", rand.Int31()) // unique error code

	defer func() {
		if err := removeMsg(bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
			fmt.Fprintf(os.Stderr, "[!:%s] remove: %s\n", ecode, err)
			somethingWrong(bot, update.Message.Chat.ID, ecode)
		}
	}()

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
	case update.CallbackQuery.Data == "wannabe" && session.Stage == stageWelcome:
		if err := sendQuizMessage(bot, dbase, update.CallbackQuery.Message.Chat.ID); err != nil {
			fmt.Fprintf(os.Stderr, "[!:%s] wannabe: %s\n", ecode, err)
			somethingWrong(bot, update.CallbackQuery.Message.Chat.ID, ecode)
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

func somethingWrong(bot *tgbotapi.BotAPI, chatID int64, ecode string) {
	text := fmt.Sprintf("%s: код %s", FatalSomeThingWrong, ecode)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown

	if _, err := bot.Send(msg); err != nil {
		fmt.Fprintf(os.Stderr, "[!:%s] SOMETHING WRONG: %s\n", ecode, err)
	}
}

func sendWelcomeMessage(bot *tgbotapi.BotAPI, dbase *badger.DB, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, MsgWelcome)
	msg.ReplyMarkup = wannabeKeyboard
	msg.ParseMode = tgbotapi.ModeMarkdown

	newMsg, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(dbase, newMsg.Chat.ID, newMsg.MessageID, stageWelcome)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

func sendQuizMessage(bot *tgbotapi.BotAPI, dbase *badger.DB, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, MsgQuiz)
	msg.ParseMode = tgbotapi.ModeMarkdown

	newMsg, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(dbase, newMsg.Chat.ID, newMsg.MessageID, stageQuiz)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

func sendAttestationAssignedMessage(bot *tgbotapi.BotAPI, dbase *badger.DB, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, MsgAttestationAssigned)
	msg.ParseMode = tgbotapi.ModeMarkdown

	newMsg, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(dbase, newMsg.Chat.ID, newMsg.MessageID, stageWait)
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

const (
	stageZero int = iota
	stageWelcome
	stageQuiz
	stageWait
	stageCleanup
	stageNone = -1

	sessionSalt = "$Rit5"
)

func sessionID(chatID int64) []byte {
	var int64bytes [8]byte

	binary.BigEndian.PutUint64(int64bytes[:], uint64(chatID))

	digest := sha256.Sum256(int64bytes[:])
	id := append([]byte{'s'}, append([]byte(sessionSalt), digest[:]...)...)

	return id
}

func setSession(dbase *badger.DB, chatID int64, msgID int, stage int) error {
	session := &Session{
		OurMsgID:   msgID,
		Stage:      stage,
		UpdateTime: time.Now().Unix(),
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	key := sessionID(chatID)
	err = dbase.Update(func(txn *badger.Txn) error {
		err := txn.Set(key, data)
		if err != nil {
			return fmt.Errorf("set: %w", err)
		}

		return nil
	})

	return nil
}

func checkSession(dbase *badger.DB, chatID int64) (*Session, error) {
	var (
		data    []byte
		session *Session = &Session{Stage: stageNone}
	)

	key := sessionID(chatID)
	err := dbase.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}

			return fmt.Errorf("get: %w", err)
		}

		err = item.Value(func(v []byte) error {
			data = append([]byte{}, v...)

			return nil
		})
		if err != nil {
			return fmt.Errorf("value: %w", err)
		}

		return nil
	})
	if err != nil {
		return session, fmt.Errorf("db: %w", err)
	}

	if data != nil {
		err := json.Unmarshal(data, session)
		if err != nil {
			return session, fmt.Errorf("parse: %w", err)
		}

		return session, nil
	}

	return session, nil
}
