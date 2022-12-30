package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/vpngen/embassy-tgbot/logs"
)

// Security chat settings - autodelete ranges.
const (
	secondsInTheDay  = 24 * 3600
	minSecondsToLive = secondsInTheDay
	maxSecondsToLive = 2 * secondsInTheDay
)

const (
	stageStart int = iota //nolint
	stageWait4Choice
	stageWait4Bill
	stageWait4Decision
	stageCleanup
)

// SlowAnswerTimeout - timeout befor each our answer.
const SlowAnswerTimeout = 3 * time.Second

// handlers options.
type hOpts struct {
	wg    *sync.WaitGroup
	db    *badger.DB
	bot   *tgbotapi.BotAPI
	cw    *ChatsWins
	debug int
}

// Handling messages (opposed callback).
func messageHandler(opts hOpts, update tgbotapi.Update) {
	defer opts.wg.Done()

	ecode := genEcode() // unique e-code

	if update.Message.ForwardFrom != nil ||
		update.Message.ForwardFromChat != nil {
		SendProtectMessage(opts.bot, update.Message.Chat.ID, 0, WarnForbidForwards, ecode)

		return
	}

	// check all dialog conditions.
	session, ok := auth(opts, update.Message.Chat.ID, update.Message.Date, ecode)
	if !ok {
		return
	}

	defer opts.cw.Release(update.Message.Chat.ID)

	if session.Stage != stageStart && update.Message.IsCommand() {
		err := handleCommands(opts, update.Message, session, ecode)
		if err != nil {
			stWrong(opts.bot, update.Message.Chat.ID, ecode, fmt.Errorf("command: %s: %w", update.Message.Command(), err))
		}

		return
	}

	// don't be in a harry.
	time.Sleep(SlowAnswerTimeout)

	switch session.Stage {
	case stageWait4Choice, stageWait4Decision, stageCleanup:
		return

	case stageWait4Bill:
		err := checkBillMessageMessage(opts, update.Message, ecode)
		if err != nil {
			stWrong(opts.bot, update.Message.Chat.ID, ecode, fmt.Errorf("bill recv: %w", err))
		}

	default:
		if warnAutodeleteSettings(opts, update.Message.Chat.ID, update.Message.Date, ecode) {
			err := sendWelcomeMessage(opts, update.Message.Chat.ID)
			if err != nil {
				stWrong(opts.bot, update.Message.Chat.ID, ecode, fmt.Errorf("welcome msg: %w", err))
			}
		}
	}
}

// Handling callbacks  (opposed messages).
func buttonHandler(opts hOpts, update tgbotapi.Update) {
	defer opts.wg.Done()

	ecode := genEcode() // unique error code

	/// check delete timeout and protect.
	session, ok := auth(opts, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.Date, ecode)
	if !ok {
		return
	}

	defer opts.cw.Release(update.CallbackQuery.Message.Chat.ID)

	// don't be in a harry.
	time.Sleep(SlowAnswerTimeout)

	switch {
	case update.CallbackQuery.Data == "started" && session.Stage == stageWait4Choice:
		if err := sendQuizMessage(opts, update.CallbackQuery.Message.Chat.ID, ecode); err != nil {
			stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("wannable push: %w", err))
		}

		// delete our previous message.
		defer func() {
			if err := RemoveMsg(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
				// we don't want to handle this
				logs.Errf("[!:%s] remove: %s\n", ecode, err)
			}
		}()
	case update.CallbackQuery.Data == "continue" && session.Stage == stageStart:
		err := sendWelcomeMessage(opts, update.CallbackQuery.Message.Chat.ID)
		if err != nil {
			stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("welcome msg: %w", err))
		}
	default:
	}
}

// genEcode - generate some uniq e-code.
func genEcode() string {
	return fmt.Sprintf("%04x", rand.Int31()) //nolint
}

// RemoveMsg - emoving message.
func RemoveMsg(bot *tgbotapi.BotAPI, chatID int64, msgID int) error {
	remove := tgbotapi.NewDeleteMessage(chatID, msgID)
	if _, err := bot.Request(remove); err != nil {
		return fmt.Errorf("request: %w", err)
	}

	return nil
}

// Something wrong handling.
func stWrong(bot *tgbotapi.BotAPI, chatID int64, ecode string, err error) {
	text := fmt.Sprintf("%s: код %s", FatalSomeThingWrongWithLink, ecode)

	logs.Debugf("[!:%s] %s\n", ecode, err)
	SendProtectMessage(bot, chatID, 0, text, ecode)
}

// Send Welcome message.
func sendWelcomeMessage(opts hOpts, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, MsgWelcome)
	msg.ReplyMarkup = WannabeKeyboard
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ProtectContent = true

	newMsg, err := opts.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(opts.db, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stageWait4Choice)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

// Send Quiz message.
func sendQuizMessage(opts hOpts, chatID int64, ecode string) error {
	msg, err := SendProtectMessage(opts.bot, chatID, 0, MsgQuiz, ecode)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(opts.db, msg.Chat.ID, msg.MessageID, int64(msg.Date), stageWait4Bill)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

// Check bill message.
func checkBillMessageMessage(opts hOpts, Message *tgbotapi.Message, ecode string) error {
	if len(Message.Photo) == 0 {
		_, err := SendProtectMessage(opts.bot, Message.Chat.ID, Message.MessageID, WarnRequiredPhoto, ecode)

		return err
	}

	photoIDX := 0
	w := 0
	for i := range Message.Photo {
		if Message.Photo[i].Width > w {
			photoIDX = i
		}
	}

	logs.Debugf("photo ID: %s\n", Message.Photo[photoIDX].FileID)

	if err := PutReceipt(opts.db, Message.Chat.ID, Message.Photo[photoIDX].FileID); err != nil {
		return fmt.Errorf("put: %w", err)
	}

	newMsg, err := SendProtectMessage(opts.bot, Message.Chat.ID, Message.MessageID, MsgAttestationAssigned, ecode)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	if err := setSession(opts.db, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stageWait4Decision); err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

// Check autodelete chat option.
func checkChatAutodeleteTimer(bot *tgbotapi.BotAPI, chatID int64) (bool, error) {
	chat, err := bot.GetChat(
		tgbotapi.ChatInfoConfig{
			ChatConfig: tgbotapi.ChatConfig{
				ChatID: chatID,
			},
		},
	)
	if err != nil {
		return false, fmt.Errorf("get chat: %w", err)
	}

	if chat.MessageAutoDeleteTime < minSecondsToLive || chat.MessageAutoDeleteTime > maxSecondsToLive {
		return false, nil
	}

	return true, nil
}

// authentificate for dilog.
func auth(opts hOpts, chatID int64, ut int, ecode string) (*Session, bool) {
	/// check session.
	session, err := checkSession(opts.db, chatID)
	if err != nil {
		stWrong(opts.bot, chatID, ecode, fmt.Errorf("check session: %w", err))

		return nil, false
	}

	if session.UpdateTime > int64(ut) {
		logs.Debugf("[!:%s] old message: %d < %d\n", ecode, session.UpdateTime, ut)

		return nil, false
	}

	if opts.cw.Get(chatID) > 0 {
		return nil, false
	}

	// show something in status.
	ca := tgbotapi.NewChatAction(chatID, getAction())
	if _, err := opts.bot.Request(ca); err != nil {
		logs.Debugf("[!:%s] chat: %s\n", ecode, err)
	}

	return session, true
}

// check autodelete.
func warnAutodeleteSettings(opts hOpts, chatID int64, ut int, ecode string) bool {
	adSet, err := checkChatAutodeleteTimer(opts.bot, chatID)
	if err != nil {
		stWrong(opts.bot, chatID, ecode, fmt.Errorf("check autodelete: %w", err))

		return false
	}

	if !adSet {
		msg := tgbotapi.NewMessage(chatID, FatalUnwellSecurity)
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ProtectContent = true
		msg.ReplyMarkup = ContinueKeyboard

		_, err := opts.bot.Send(msg)
		if err != nil {
			logs.Errf("[!:%s] send message: %s\n", ecode, err)
		}

		return false
	}

	return true
}

func getAction() string {
	ix := int(rand.Int31n(int32(len(StandartChatActions)))) //nolint

	return StandartChatActions[ix]
}

func handleCommands(opts hOpts, Message *tgbotapi.Message, session *Session, ecode string) error {
	switch Message.Command() {
	case "reset":
		if opts.debug == int(logs.LevelDebug) {
			err := resetSession(opts.db, Message.Chat.ID)
			if err == nil {
				SendProtectMessage(opts.bot, Message.Chat.ID, 0, ResetSuccessfull, ecode)
			}

			return err
		}

		fallthrough
	default:
		SendProtectMessage(opts.bot, Message.Chat.ID, 0, WarnUnknownCommand, ecode)

		return nil
	}
}

func SendOpenMessage(bot *tgbotapi.BotAPI, chatID int64, replyID int, text, ecode string) (*tgbotapi.Message, error) {
	msg, err := sendMessage(bot, chatID, replyID, false, text, ecode)
	if err != nil {
		return msg, fmt.Errorf("open msg: %w", err)
	}

	return msg, nil
}

func SendProtectMessage(bot *tgbotapi.BotAPI, chatID int64, replyID int, text, ecode string) (*tgbotapi.Message, error) {
	msg, err := sendMessage(bot, chatID, replyID, true, text, ecode)
	if err != nil {
		return msg, fmt.Errorf("protect msg: %w", err)
	}

	return msg, nil
}

// SendMessage - send common message.
func sendMessage(bot *tgbotapi.BotAPI, chatID int64, replyID int, protect bool, text, ecode string) (*tgbotapi.Message, error) {
	logs.Debugf("[!:%s] send message\n", ecode)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ProtectContent = protect

	if replyID != 0 {
		msg.ReplyToMessageID = replyID
	}

	newMsg, err := bot.Send(msg)
	if err != nil {
		logs.Errf("[!:%s] send message: %s\n", ecode, err)

		return nil, err
	}

	return &newMsg, nil
}
