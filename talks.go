package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
	tgbotapi "github.com/vpngen/embassy-tgbot/telegram-bot-api"

	"github.com/vpngen/embassy-tgbot/logs"
)

// Security chat settings - autodelete ranges.
const (
	secondsInTheDay  = 24 * 3600
	minSecondsToLive = secondsInTheDay
	maxSecondsToLive = 3 * secondsInTheDay
)

const (
	stageMainTrackStart int = iota //nolint
	stageMainTrackWaitForWanting
	stageMainTrackWaitForBill
	stageMainTrackWaitForApprovement
	stageMainTrackCleanup
	stageRestoreTrackStart     // user apply /restore command
	stageRestoreTrackSendName  // user send brigadier name
	stageRestoreTrackSendWords // user send seed words
	stageRestoreTrackCleanup   // user received config
)

// SlowAnswerTimeout - timeout befor each our answer.
const SlowAnswerTimeout = 3 * time.Second

// handlers options.
type handlerOpts struct {
	wg               *sync.WaitGroup
	db               *badger.DB
	bot              *tgbotapi.BotAPI
	cw               *ChatsWins
	debug            int
	ls               *LabelStorage
	mnt              *Maintenance
	supportURL       string
	flowMainUrl      string
	flowDecisionsUrl string

	sessionSecret []byte
	queueSecret   []byte
}

var onlyBase64Symbols = regexp.MustCompile(`[^A-Za-z0-9\-_]`)

func IsForbiddenError(err error) bool {
	tgErr := &tgbotapi.Error{}
	if errors.As(err, &tgErr) {
		if tgErr.Code == 403 {
			return true
		}
	}

	return false
}

// Handling reaction (opposed callback).
func reactionHandler(opts handlerOpts, update tgbotapi.Update) {
	defer opts.wg.Done()

	ecode := genEcode() // unique e-code

	if update.MessageReaction.Chat.Type != "private" {
		SendProtectedMessage(opts.bot, update.MessageReaction.Chat.ID, 0, false, InfoForbidForwardsMessage, ecode)

		return
	}

	// check all dialog conditions.
	session, ok := auth(opts, update.MessageReaction.Chat.ID, update.MessageReaction.Date, ecode)
	if !ok {
		return
	}

	defer opts.cw.Release(update.MessageReaction.Chat.ID)

	// don't be in a harry.
	time.Sleep(SlowAnswerTimeout)

	c := session.Captcha

	if !c.Passed && c.MessageID == update.MessageReaction.MessageID {
		for _, r := range update.MessageReaction.NewReaction {
			if r.Type == "emoji" && r.Emoji == c.Reaction {
				c.Passed = true

				if err := sendSuccessLike(opts, session.Label, &c, update.MessageReaction.Chat.ID, session.Stage, session.State); err != nil {
					stWrong(opts.bot, update.MessageReaction.Chat.ID, ecode, fmt.Errorf("success like: %w", err))
				}

				break
			}
		}
	}
}

// Send Success like message.
func sendSuccessLike(opts handlerOpts, label SessionLabel, c *SessionCaptcha, chatID int64, stage, state int) error {
	msg := tgbotapi.NewMessage(chatID, "Проверка пройдена! Нажми /repeate для продолжения.")
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = true

	newMsg, err := opts.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(opts.db, opts.sessionSecret, label, c, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stage, state, nil)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

// Handling messages (opposed callback).
func messageHandler(opts handlerOpts, update tgbotapi.Update, dept MinistryOpts) {
	defer opts.wg.Done()

	ecode := genEcode() // unique e-code

	lang := userLang(update.Message.From.LanguageCode)

	if update.Message.ForwardFrom != nil ||
		update.Message.ForwardFromChat != nil {
		SendProtectedMessage(opts.bot, update.Message.Chat.ID, 0, false, InfoForbidForwardsMessage, ecode)

		return
	}

	// check all dialog conditions.
	session, ok := auth(opts, update.Message.Chat.ID, update.Message.Date, ecode)
	if !ok {
		return
	}

	defer opts.cw.Release(update.Message.Chat.ID)

	// don't be in a harry.
	time.Sleep(SlowAnswerTimeout)

	if update.Message.IsCommand() {
		err := handleCommands(opts, update.Message, session, dept, ecode, lang)
		if err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.Message.Chat.ID, ecode, fmt.Errorf("command: %s: %w", update.Message.Command(), err))
		}

		return
	}

	switch session.Stage {
	case stageMainTrackCleanup:
		_, err := SendProtectedMessage(opts.bot, update.Message.Chat.ID, update.Message.MessageID, false, MainTrackWarnConversationsFinished, ecode)
		if err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.Message.Chat.ID, ecode, fmt.Errorf("end msg: %w", err))
		}
	case stageMainTrackWaitForApprovement:
		if checkMaintenanceMode(opts, session.Label, &session.Captcha, update.Message.Chat.ID, ecode, false) {
			return
		}

		_, err := SendProtectedMessage(opts.bot, update.Message.Chat.ID, update.Message.MessageID, false, MainTrackWarnWaitForApprovement, ecode)
		if err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.Message.Chat.ID, ecode, fmt.Errorf("wait msg: %w", err))
		}
	case stageMainTrackWaitForBill:
		if checkMaintenanceMode(opts, session.Label, &session.Captcha, update.Message.Chat.ID, ecode, false) {
			return
		}

		err := checkBillMessageMessage(opts, session.Label, &session.Captcha, update.Message, ecode, lang)
		if err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.Message.Chat.ID, ecode, fmt.Errorf("bill recv: %w", err))
		}
	case stageRestoreTrackStart:
		if checkMaintenanceMode(opts, session.Label, &session.Captcha, update.Message.Chat.ID, ecode, true) {
			return
		}

		if err := sendRestoreStartMessage(opts, session.Label, &session.Captcha, update.Message.Chat.ID, session.State, lang); err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.Message.Chat.ID, ecode, fmt.Errorf("start restore push: %w", err))
		}
	case stageRestoreTrackSendName:
		if checkMaintenanceMode(opts, session.Label, &session.Captcha, update.Message.Chat.ID, ecode, true) {
			return
		}

		defer func() {
			if session.OurMsgID == 0 {
				return
			}

			if err := RemoveKeyboardMsg(opts.bot, update.Message.Chat.ID, session.OurMsgID, RestoreTrackNameMessage); err != nil {
				// we don't want to handle this
				logs.Errf("[!:%s] remove keyboard: %s\n", ecode, err)
			}
		}()

		err := checkRestoreNameMessageMessage(opts, session.Label, &session.Captcha, update.Message, session.State, lang)
		if err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.Message.Chat.ID, ecode, fmt.Errorf("name recv: %w", err))
		}
	case stageRestoreTrackSendWords:
		if checkMaintenanceMode(opts, session.Label, &session.Captcha, update.Message.Chat.ID, ecode, true) {
			return
		}

		defer func() {
			if session.OurMsgID == 0 {
				return
			}

			if err := RemoveKeyboardMsg(opts.bot, update.Message.Chat.ID, session.OurMsgID, RestoreTrackWordsMessage); err != nil {
				// we don't want to handle this
				logs.Errf("[!:%s] remove keyboard: %s\n", ecode, err)
			}
		}()

		err := checkRestoreWordsMessageMessage(opts, session.Label, &session.Captcha, update.Message, ecode, session.State, session.Payload, dept, lang)
		if err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.Message.Chat.ID, ecode, fmt.Errorf("words recv: %w", err))
		}
	case stageMainTrackWaitForWanting:
		fallthrough
	default:
		if warnAutodeleteSettings(opts, update.Message.Chat.ID, ecode) {

			fmt.Fprintf(os.Stderr, "new session: %#v\n", session)

			session.Label = setLabel(session.Label, MarkerEmptyLabel)

			if !checkCaptcha(opts, &session.Captcha, session.Label, update.Message.Chat.ID, ecode, session.Stage, session.State, session.Payload) {
				return
			}

			err := sendWelcomeMessage(opts, session.Label, &session.Captcha, update.Message.Chat.ID, lang)
			if err != nil {
				if IsForbiddenError(err) {
					setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

					return
				}

				stWrong(opts.bot, update.Message.Chat.ID, ecode, fmt.Errorf("welcome msg: %w", err))
			}

			return
		}

	}
}

// Handling callbacks  (opposed messages).
func buttonHandler(opts handlerOpts, update tgbotapi.Update, dept MinistryOpts) {
	defer opts.wg.Done()

	ecode := genEcode() // unique error code

	lang := userLang(update.CallbackQuery.From.LanguageCode)

	/// check delete timeout and protect.
	session, ok := auth(opts, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.Date, ecode)
	if !ok {
		return
	}

	defer opts.cw.Release(update.CallbackQuery.Message.Chat.ID)

	// don't be in a harry.
	time.Sleep(SlowAnswerTimeout)

	switch {
	case update.CallbackQuery.Data == "started" && session.Stage == stageMainTrackWaitForWanting:
		if checkMaintenanceMode(opts, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, ecode, false) {
			return
		}

		if err := sendQuizMessage(opts, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, ecode, lang); err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("wannable push: %w", err))
		}

		// delete our previous message.
		defer func() {
			if err := RemoveMsg(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
				// we don't want to handle this
				logs.Errf("[!:%s] remove: %s\n", ecode, err)
			}
		}()
	case update.CallbackQuery.Data == "continue" && (session.Stage == stageMainTrackStart || session.Stage == stageMainTrackWaitForWanting):
		if checkMaintenanceMode(opts, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, ecode, false) {
			return
		}

		if !checkCaptcha(opts, &session.Captcha, session.Label, update.CallbackQuery.Message.Chat.ID, ecode, session.Stage, session.State, session.Payload) {
			return
		}

		err := sendWelcomeMessage(opts, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, lang)
		if err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("welcome msg: %w", err))
		}
	case update.CallbackQuery.Data == "restore" && session.Stage == stageRestoreTrackStart:
		if checkMaintenanceMode(opts, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, ecode, true) {
			return
		}

		if err := sendRestoreNameMessage(opts, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, session.State, lang); err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("restore push: %w", err))
		}

		// delete our previous message.
		defer func() {
			if err := RemoveMsg(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
				// we don't want to handle this
				logs.Errf("[!:%s] remove: %s\n", ecode, err)
			}
		}()
	case session.Stage == stageRestoreTrackSendWords &&
		(update.CallbackQuery.Data == "again" || update.CallbackQuery.Data == "return"):
		if checkMaintenanceMode(opts, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, ecode, false) {
			return
		}

		defer func() {
			text := RestoreTrackWordsMessage

			switch update.CallbackQuery.Data {
			case "again":
				text = RestoreTrackBrigadeNotFoundMessage
			case "retrun":
				text = RestoreTrackWordsMessage
			}
			if err := RemoveKeyboardMsg(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, text); err != nil {
				// we don't want to handle this
				logs.Errf("[!:%s] restore keyboard: %s\n", ecode, err)
			}
		}()

		if err := sendRestoreNameMessage(opts, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, session.State, lang); err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("again push: %w", err))
		}
	case update.CallbackQuery.Data == "vip":
		label := ""
		sessionLabel := ""

		x := rand.Intn(len(MainTrackQuizMessage))
		for prefix := range MainTrackQuizMessage {
			if x == 0 {
				label = prefix + label
				if len(label) > 64 {
					label = label[:64]
				}

				sessionLabel = label

				break
			}

			x--
		}

		if session.Label.Time.IsZero() || session.Label.ID == uuid.Nil {
			session.Label = SessionLabel{
				Label: sessionLabel,
				Time:  time.Now(),
				ID:    uuid.New(),
			}
		}

		if session.Label.Label == "" {
			session.Label.Label = sessionLabel
		}

		requestID, err := reqBrigade(dept, update.CallbackQuery.Message.Chat.ID, session.Label, "")
		if err != nil || requestID == uuid.Nil {
			stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("request brigade failed"))

			return
		}

		if err := sendVIPMessage(opts, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, requestID, lang); err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("vip push: %w", err))
		}

		// delete our previous message.
		defer func() {
			if err := RemoveMsg(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
				// we don't want to handle this
				logs.Errf("[!:%s] remove: %s\n", ecode, err)
			}
		}()
	case update.CallbackQuery.Data == "reset":
		if session.State == SessionStatePayloadSecondary {
			if _, err := SendProtectedMessage(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, false, MainTrackWarnConversationsFinished, ecode); err != nil {
				if IsForbiddenError(err) {
					setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

					return
				}

				stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("end msg: %w", err))

				return
			}
		}

		fmt.Fprintf(os.Stderr, "reset session: %#v\n", session)

		session.Label = setLabel(session.Label, MarkerResetLabel)

		if err := sendWelcomeMessage(opts, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, lang); err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("reset push: %w", err))

			return
		}

		// delete our previous message.
		defer func() {
			if err := RemoveMsg(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
				// we don't want to handle this
				logs.Errf("[!:%s] remove: %s\n", ecode, err)
			}
		}()
	case update.CallbackQuery.Data == "outline_download_urls":
		if err := sendDownloadOutlineMessage(opts.bot, update.CallbackQuery.Message.Chat.ID, lang, opts.flowMainUrl); err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("end msg: %w", err))
		}
	case update.CallbackQuery.Data == "amnezia_vpn_download_urls":
		if err := sendDownloadAmneziaVPNMessage(opts.bot, update.CallbackQuery.Message.Chat.ID, opts.flowMainUrl); err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("end msg: %w", err))
		}
	case update.CallbackQuery.Data == "restore":
		if checkMaintenanceMode(opts, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, ecode, true) {
			return
		}

		prev := 0
		if session.Stage == stageMainTrackCleanup || session.Stage == stageRestoreTrackStart ||
			session.Stage == stageRestoreTrackSendName || session.Stage == stageRestoreTrackSendWords ||
			session.Stage == stageRestoreTrackCleanup {
			prev = session.State
		}

		if prev == SessionStatePayloadBan {
			_, err := SendProtectedMessage(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, false, MainTrackWarnConversationsFinished, ecode)
			if err != nil {
				if IsForbiddenError(err) {
					setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

					return
				}

				stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("end msg: %w", err))
			}

			return
		}

		if err := sendRestoreStartMessage(opts, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, prev, lang); err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, update.CallbackQuery.Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return
			}

			stWrong(opts.bot, update.CallbackQuery.Message.Chat.ID, ecode, fmt.Errorf("restore push: %w", err))
		}

		// delete our previous message.
		defer func() {
			if err := RemoveMsg(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
				// we don't want to handle this
				logs.Errf("[!:%s] remove: %s\n", ecode, err)
			}
		}()
	default:
		fmt.Fprintf(os.Stderr, "unknown callback: %q session: %#v\n", update.CallbackQuery.Data, session)
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

// RemoveKeyboardMsg - emoving message.
func RemoveKeyboardMsg(bot *tgbotapi.BotAPI, chatID int64, msgID int, text string) error {
	msg := tgbotapi.NewEditMessageText(chatID, msgID, text)
	msg.ReplyMarkup = nil
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := bot.Request(msg); err != nil {
		return fmt.Errorf("request: %w", err)
	}

	return nil
}

// Something wrong handling.
func stWrong(bot *tgbotapi.BotAPI, chatID int64, ecode string, err error) {
	text := fmt.Sprintf("%s: код %s", FatalSomeThingWrong, ecode)

	logs.Debugf("[!:%s] %s\n", ecode, err)
	SendProtectedMessage(bot, chatID, 0, false, text, ecode)
}

// Send Welcome message.
func sendWelcomeMessage(opts handlerOpts, label SessionLabel, c *SessionCaptcha, chatID int64, lang string) error {
	// OLD:
	// msg := tgbotapi.NewMessage(chatID, MainTrackWelcomeMessage)
	// msg.ReplyMarkup = WannabeKeyboard
	msg := tgbotapi.NewMessage(chatID, flowMessage(opts.flowMainUrl, "welcome", MainTrackWelcomeMessage, lang))
	if kb, ok := flowKeyboard(opts.flowMainUrl, "welcome", opts.supportURL, lang); ok {
		msg.ReplyMarkup = *kb
	} else {
		msg.ReplyMarkup = WannabeKeyboard
	}
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = true

	newMsg, err := opts.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(opts.db, opts.sessionSecret, label, c, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stageMainTrackWaitForWanting, SessionStatePayloadSomething, nil)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

// Send Welcome message.
func sendVIPMessage2(opts handlerOpts, label SessionLabel, c *SessionCaptcha, chatID int64, lang string) error {
	msg := tgbotapi.NewMessage(chatID, flowMessage(opts.flowMainUrl, "vip_welcome", MainTrackVIPWelcomeMessage, lang))
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = true

	newMsg, err := opts.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(opts.db, opts.sessionSecret, label, c, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stageMainTrackStart, SessionStatePayloadSomething, nil)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

// Send VIP message.
func sendVIPMessage(opts handlerOpts, label SessionLabel, c *SessionCaptcha, chatID int64, requestID uuid.UUID, lang string) error {
	msg := tgbotapi.NewMessage(chatID, flowMessage(opts.flowMainUrl, "vip", VIPMessage, lang))

	vipURL := VIPBotURL + "?start=" + requestID.String()
	if kb, ok := flowKeyboard(opts.flowMainUrl, "vip", opts.supportURL, lang, map[string]string{"vip_bot_url": vipURL}); ok {
		msg.ReplyMarkup = *kb
	} else {
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("Перейти в VIP-бот", vipURL),
				tgbotapi.NewInlineKeyboardButtonData("Передумал", "reset")),
		)
	}
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = true

	newMsg, err := opts.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(opts.db, opts.sessionSecret, label, c, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stageMainTrackStart, SessionStatePayloadSomething, nil)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

// Send Quiz message.
func sendQuizMessage(opts handlerOpts, label SessionLabel, c *SessionCaptcha, chatID int64, ecode string, lang string) error {
	num, _, _ := strings.Cut(label.Label, "_")

	text := MainTrackQuizMessage[num+"_"]
	if text == "" {
		for prefix := range MainTrackQuizMessage {
			text = MainTrackQuizMessage[prefix]

			break
		}
	}

	// OLD: text was selected only from MainTrackQuizMessage map.
	text = flowMessage(opts.flowMainUrl, "quiz", text, lang)

	msg, err := SendProtectedMessage(opts.bot, chatID, 0, false, text, ecode)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(opts.db, opts.sessionSecret, label, c, msg.Chat.ID, msg.MessageID, int64(msg.Date), stageMainTrackWaitForBill, SessionStatePayloadSomething, nil)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

const MainTrackWeAreSoBusy = `Прости нас, неимоверная нагрузка на ресурсы. Сами расстроены. Повтори попытку позже 🤷‍♂️`

// Check bill message.
func checkBillMessageMessage(opts handlerOpts, label SessionLabel, c *SessionCaptcha, Message *tgbotapi.Message, ecode string, lang string) error {
	if len(Message.Photo) == 0 {
		_, err := SendProtectedMessage(opts.bot, Message.Chat.ID, Message.MessageID, false, MainTrackWarnRequiredPhoto, ecode)

		return err
	}

	_, _, count, err := catchFirstReceipt(opts.db, CkReceiptStageNone)
	if err != nil {
		return fmt.Errorf("catch: %w", err)
	}

	if count > 10 {
		SendProtectedMessage(opts.bot, Message.Chat.ID, Message.MessageID, false, MainTrackWeAreSoBusy, ecode)

		return nil
	}

	photoIDX := 0
	w := 0
	for i := range Message.Photo {
		if Message.Photo[i].Width > w {
			photoIDX = i
		}
	}

	logs.Debugf("photo ID: %s\n", Message.Photo[photoIDX].FileID)

	if err := PutReceipt(opts.db, opts.queueSecret, Message.Chat.ID, Message.Photo[photoIDX].FileID, lang); err != nil {
		return fmt.Errorf("put: %w", err)
	}

	newMsg, err := SendProtectedMessage(opts.bot, Message.Chat.ID, Message.MessageID, false, MainTrackSendForAttestationMessage, ecode)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	if err := setSession(opts.db, opts.sessionSecret, label, c, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stageMainTrackWaitForApprovement, SessionStatePayloadSomething, nil); err != nil {
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

// Send Start Restore message.
func sendRestoreStartMessage(opts handlerOpts, label SessionLabel, c *SessionCaptcha, chatID int64, prev int, lang string) error {
	// OLD:
	// msg := tgbotapi.NewMessage(chatID, RestoreTrackStartMessage)
	// msg.ReplyMarkup = RestoreStartKeyboard
	msg := tgbotapi.NewMessage(chatID, flowMessage(opts.flowMainUrl, "restore_start", RestoreTrackStartMessage, lang))
	if kb, ok := flowKeyboard(opts.flowMainUrl, "restore_start", opts.supportURL, lang); ok {
		msg.ReplyMarkup = *kb
	} else {
		msg.ReplyMarkup = RestoreStartKeyboard
	}
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = true

	newMsg, err := opts.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(opts.db, opts.sessionSecret, label, c, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stageRestoreTrackStart, prev, nil)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

// Send Name message.
func sendRestoreNameMessage(opts handlerOpts, label SessionLabel, c *SessionCaptcha, chatID int64, prev int, lang string) error {
	// OLD:
	// msg := tgbotapi.NewMessage(chatID, RestoreTrackNameMessage)
	// msg.ReplyMarkup = RestoreNameKeyboard
	msg := tgbotapi.NewMessage(chatID, flowMessage(opts.flowMainUrl, "restore_name", RestoreTrackNameMessage, lang))
	if kb, ok := flowKeyboard(opts.flowMainUrl, "restore_name", opts.supportURL, lang); ok {
		msg.ReplyMarkup = *kb
	} else {
		msg.ReplyMarkup = RestoreNameKeyboard
	}
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = true

	newMsg, err := opts.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	err = setSession(opts.db, opts.sessionSecret, label, c, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stageRestoreTrackSendName, prev, nil)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

func sendRestoreWordsMessage(opts handlerOpts, label SessionLabel, c *SessionCaptcha, chatID int64, prev int, text string, lang string) error {
	// OLD:
	// msg := tgbotapi.NewMessage(chatID, RestoreTrackWordsMessage)
	// msg.ReplyMarkup = RestoreWordsKeyboard1
	msg := tgbotapi.NewMessage(chatID, flowMessage(opts.flowMainUrl, "restore_words", RestoreTrackWordsMessage, lang))
	if kb, ok := flowKeyboard(opts.flowMainUrl, "restore_words", opts.supportURL, lang); ok {
		msg.ReplyMarkup = *kb
	} else {
		msg.ReplyMarkup = RestoreWordsKeyboard1
	}
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = true

	newMsg, err := opts.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	if err := setSession(opts.db, opts.sessionSecret, label, c, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stageRestoreTrackSendWords, prev, []byte(text)); err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return err
}

// Check restore name message.
func checkRestoreNameMessageMessage(opts handlerOpts, label SessionLabel, c *SessionCaptcha, Message *tgbotapi.Message, prev int, lang string) error {
	text := strings.Join(
		strings.Fields(
			strings.TrimSpace(
				strings.Replace(
					strings.Replace(Message.Text, ",", " ", -1),
					"\"", "", -1),
			),
		),
		" ",
	)

	_, _, ok := strings.Cut(text, " ")
	if !ok || !utf8.ValidString(text) {
		msg := tgbotapi.NewMessage(Message.Chat.ID, RestoreTrackInvalidNameMessageVIP)
		msg.ReplyMarkup = RestoreNameKeyboard
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.DisableWebPagePreview = true
		msg.ProtectContent = true

		newMsg, err := opts.bot.Send(msg)
		if err != nil {
			return fmt.Errorf("send: %w", err)
		}

		err = setSession(opts.db, opts.sessionSecret, label, c, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stageRestoreTrackSendName, prev, nil)
		if err != nil {
			return fmt.Errorf("session: %w", err)
		}

		return err
	}

	err := sendRestoreWordsMessage(opts, label, c, Message.Chat.ID, prev, text, lang)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return nil
}

func sendWordsFailed(opts handlerOpts, label SessionLabel, c *SessionCaptcha, chatID int64, prev int, text []byte) error {
	msg := tgbotapi.NewMessage(chatID, RestoreTrackBrigadeNotFoundMessageVIP)
	msg.ReplyMarkup = RestoreWordsKeyboard2
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = true

	newMsg, err := opts.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	if err := setSession(opts.db, opts.sessionSecret, label, c, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stageRestoreTrackSendWords, prev, text); err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return err
}

// Check restore words message.
func checkRestoreWordsMessageMessage(opts handlerOpts, label SessionLabel, c *SessionCaptcha, Message *tgbotapi.Message, ecode string, prev int, name []byte, dept MinistryOpts, lang string) error {
	if name == nil {
		return sendWordsFailed(opts, label, c, Message.Chat.ID, prev, nil)
	}

	words := strings.Join(
		strings.Fields(
			strings.TrimSpace(
				strings.Replace(
					strings.Replace(Message.Text, ",", " ", -1),
					"\"", "", -1),
			),
		),
		" ",
	)

	if words == "" || len(strings.Split(words, " ")) < 6 {
		return sendWordsFailed(opts, label, c, Message.Chat.ID, prev, name)
	}

	if !utf8.ValidString(words) {
		return sendWordsFailed(opts, label, c, Message.Chat.ID, prev, name)
	}

	err := RestoreBrigadier(opts.bot, Message.Chat.ID, ecode, dept, opts.mnt, string(name), words, lang, opts.flowMainUrl)
	if err != nil {
		return sendWordsFailed(opts, label, c, Message.Chat.ID, prev, name)
	}

	if err := setSession(opts.db, opts.sessionSecret, label, c, Message.Chat.ID, 0, 0, stageRestoreTrackCleanup, prev, nil); err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}

// authentificate for dilog.
func auth(opts handlerOpts, chatID int64, ut int, ecode string) (*Session, bool) {
	/// check session.
	session, err := checkSession(opts.db, opts.sessionSecret, chatID)
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
func warnAutodeleteSettings(opts handlerOpts, chatID int64, ecode string) bool {
	/*adSet, err := checkChatAutodeleteTimer(opts.bot, chatID)
	if err != nil {
		stWrong(opts.bot, chatID, ecode, fmt.Errorf("check autodelete: %w", err))

		return false
	}

	if !adSet {
		msg := tgbotapi.NewMessage(chatID, MainTrackUnwellSecurityMessage)
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ProtectContent = true
		msg.ReplyMarkup = ContinueKeyboard

		_, err := opts.bot.Send(msg)
		if err != nil {
			logs.Errf("[!:%s] send message: %s\n", ecode, err)
		}

		return false
	}*/

	return true
}

// check captcha.
func checkCaptcha(opts handlerOpts, c *SessionCaptcha, label SessionLabel, chatID int64, ecode string, stage, state int, text []byte) bool {
	return true
	/*
	   	if c == nil {
	   		c = &SessionCaptcha{}
	   	}

	   	if c.Passed {
	   		return true
	   	}

	   	if c.Attempts >= 5 {
	   		setSession(opts.db, opts.sessionSecret, label, nil, 0, 0, int64(time.Now().Unix()), stageMainTrackCleanup, SessionStatePayloadBan, nil)
	   	}

	   	if c.SleepTill.After(time.Now()) {
	   		wt := time.Until(c.SleepTill)
	   		if wt <= 0 {
	   			return false
	   		}

	   		txt := fmt.Sprintf("Не так быстро. Пожалуйста, повтори попытку через %d минут(ы)", int(math.Ceil(wt.Minutes())))
	   		msg := tgbotapi.NewMessage(chatID, txt)
	   		msg.ParseMode = tgbotapi.ModeMarkdown
	   		msg.ProtectContent = true

	   		nMsg, err := opts.bot.Send(msg)
	   		if err != nil {
	   			logs.Errf("[!:%s] send message: %s\n", ecode, err)

	   			return false
	   		}

	   		go func() {
	   			<-time.After(wt)

	   			if err := RemoveMsg(opts.bot, nMsg.Chat.ID, nMsg.MessageID); err != nil {
	   				logs.Errf("[!:%s] remove message: %s\n", ecode, err)
	   			}
	   		}()

	   		return false
	   	}

	   	if c.PrevSleep >= len(CaptchaEscalationTimes) {
	   		c.PrevSleep = 0
	   	}

	   c.SleepTill = time.Now().Add(CaptchaEscalationTimes[c.PrevSleep])
	   c.PrevSleep++
	   c.Attempts++

	   like, captchaText := GetCaptchaText()
	   c.Reaction = like

	   msg := tgbotapi.NewMessage(chatID, captchaText)
	   msg.ParseMode = tgbotapi.ModeMarkdown
	   msg.ProtectContent = true

	   newMsg, err := opts.bot.Send(msg)

	   	if err != nil {
	   		logs.Errf("[!:%s] send message: %s\n", ecode, err)
	   	}

	   c.MessageID = newMsg.MessageID

	   setSession(opts.db, opts.sessionSecret, label, c, newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date), stage, state, text)

	   // delete our previous message.

	   	go func() {
	   		<-time.After(CaptchaLivetime)

	   		if newMsg.Chat != nil {
	   			if err := RemoveMsg(opts.bot, newMsg.Chat.ID, newMsg.MessageID); err == nil {
	   				session, err := checkSession(opts.db, opts.sessionSecret, chatID)
	   				if err != nil {
	   					logs.Errf("[!:%s] check session: %s\n", ecode, err)

	   					return
	   				}

	   				if !session.Captcha.Passed {
	   					wt := time.Until(session.Captcha.SleepTill)
	   					if wt <= 0 {
	   						wt = 0
	   					}

	   					txt := fmt.Sprintf("Время истекло. Пожалуйста, повтори попытку через %d минут(ы)", int(math.Ceil(wt.Minutes())))
	   					msg := tgbotapi.NewMessage(chatID, txt)
	   					msg.ParseMode = tgbotapi.ModeMarkdown
	   					msg.ProtectContent = true

	   					nMsg, err := opts.bot.Send(msg)
	   					if err != nil {
	   						logs.Errf("[!:%s] send message: %s\n", ecode, err)

	   						return
	   					}

	   					go func() {
	   						<-time.After(wt)

	   						if err := RemoveMsg(opts.bot, nMsg.Chat.ID, nMsg.MessageID); err != nil {
	   							logs.Errf("[!:%s] remove message: %s\n", ecode, err)
	   						}
	   					}()
	   				}
	   			}
	   		}
	   	}()

	   return false
	*/
}

func getAction() string {
	ix := int(rand.Int31n(int32(len(StandardChatActions)))) //nolint

	return StandardChatActions[ix]
}

func handleCommands(opts handlerOpts, Message *tgbotapi.Message, session *Session, dept MinistryOpts, ecode string, lang string) error {
	logs.Debugf("[d:%s] stage:  %d\n", ecode, session.Stage)

	command := Message.Command()

	if opts.debug == int(logs.LevelDebug) && command == "vpnregen" {
		logs.Debugf("[d:%s] command: vpnregen (debug reset)\n", ecode)

		err := resetSession(opts.db, opts.sessionSecret, Message.Chat.ID)
		if err != nil {
			return fmt.Errorf("vpnregen: %w", err)
		}

		if _, err := SendProtectedMessage(opts.bot, Message.Chat.ID, 0, false, MainTrackResetSuccessfull, ecode); err != nil {
			return fmt.Errorf("send welcome: %w", err)
		}

		return nil
	}

	switch command {
	case "start":
		logs.Debugf("[d:%s] command: start, args: %s\n", ecode, Message.CommandArguments())

		s := Message.CommandArguments()
		logs.Debugf("[d:%s] start args: %q\n, %d", ecode, s, len(s))
		if len(s) == 36 {
			if _, err := uuid.Parse(s); err == nil {
				requestID, err := reqBrigade(dept, Message.Chat.ID, session.Label, s)
				if err != nil || requestID == uuid.Nil {
					stWrong(opts.bot, Message.Chat.ID, ecode, fmt.Errorf("request custom brigade failed"))

					return nil
				}

				if err := sendVIPMessage2(opts, session.Label, &session.Captcha, Message.Chat.ID, lang); err != nil {
					if IsForbiddenError(err) {
						setSession(opts.db, opts.sessionSecret, session.Label, &session.Captcha, Message.Chat.ID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

						return nil
					}

					stWrong(opts.bot, Message.Chat.ID, ecode, fmt.Errorf("custom vip push: %w", err))
				}

				// it's a valid uuid, but we don't want to handle this.
				return nil
			}
		}

		fallthrough
	case "restore":
		logs.Debugf("[d:%s] command: restore\n", ecode)

		if checkMaintenanceMode(opts, session.Label, &session.Captcha, Message.Chat.ID, ecode, true) {
			return nil
		}

		prev := 0
		if session.Stage == stageMainTrackCleanup || session.Stage == stageRestoreTrackStart ||
			session.Stage == stageRestoreTrackSendName || session.Stage == stageRestoreTrackSendWords ||
			session.Stage == stageRestoreTrackCleanup {
			prev = session.State
		}

		if prev == SessionStatePayloadBan {
			_, err := SendProtectedMessage(opts.bot, Message.Chat.ID, Message.MessageID, false, MainTrackWarnConversationsFinished, ecode)
			if err != nil {
				return fmt.Errorf("end msg: %w", err)
			}

			return nil
		}

		time.Sleep(SlowAnswerTimeout)

		if err := sendRestoreStartMessage(opts, session.Label, &session.Captcha, Message.Chat.ID, prev, lang); err != nil {
			return fmt.Errorf("restore msg: %w", err)
		}

		return nil
	case "repeat":
		logs.Debugf("[d:%s] command: repeat, stage: %d\n", ecode, session.Stage)

		if checkMaintenanceMode(opts, session.Label, &session.Captcha, Message.Chat.ID, ecode, true) {
			return nil
		}

		switch session.Stage {
		case stageMainTrackWaitForBill:
			logs.Debugf("[d:%s] command: repeat -> stageMainTrackWaitForBill\n", ecode)
			if err := sendQuizMessage(opts, session.Label, &session.Captcha, Message.Chat.ID, ecode, lang); err != nil {
				return fmt.Errorf("wait for bill: %w", err)
			}

			return nil
		case stageMainTrackCleanup:
			logs.Debugf("[d:%s] command: repeat -> stageMainTrackCleanup\n", ecode)

			_, err := SendProtectedMessage(opts.bot, Message.Chat.ID, Message.MessageID, false, RepeatTrackWarnConversationsFinished, ecode)
			if err != nil {
				return fmt.Errorf("end msg: %w", err)
			}

			return nil
		}

		fallthrough // !!! it'a a dirty hack. we neeed rewrite this code.
	default:
		logs.Debugf("[d:%s] command: %s (default), stage: %d\n", ecode, command, session.Stage)

		switch session.Stage {
		case stageRestoreTrackSendWords:
			logs.Debugf("[d:%s] default -> stageRestoreTrackSendWords\n", ecode)
			if checkMaintenanceMode(opts, session.Label, &session.Captcha, Message.Chat.ID, ecode, true) {
				return nil
			}

			if err := sendRestoreWordsMessage(opts, session.Label, &session.Captcha, Message.Chat.ID, session.State, string(session.Payload), lang); err != nil {
				return fmt.Errorf("send words: %w", err)
			}
		case stageRestoreTrackSendName:
			logs.Debugf("[d:%s] default -> stageRestoreTrackSendName\n", ecode)

			if checkMaintenanceMode(opts, session.Label, &session.Captcha, Message.Chat.ID, ecode, true) {
				return nil
			}

			if err := sendRestoreNameMessage(opts, session.Label, &session.Captcha, Message.Chat.ID, session.State, lang); err != nil {
				return fmt.Errorf("send name: %w", err)
			}
		case stageRestoreTrackStart:
			logs.Debugf("[d:%s] default -> stageRestoreTrackStart\n", ecode)

			if checkMaintenanceMode(opts, session.Label, &session.Captcha, Message.Chat.ID, ecode, true) {
				return nil
			}

			if err := sendRestoreStartMessage(opts, session.Label, &session.Captcha, Message.Chat.ID, session.State, lang); err != nil {
				return fmt.Errorf("restore: %w", err)
			}
		case stageMainTrackCleanup:
			logs.Debugf("[d:%s] default -> stageMainTrackCleanup\n", ecode)
			_, err := SendProtectedMessage(opts.bot, Message.Chat.ID, Message.MessageID, false, MainTrackWarnConversationsFinished, ecode)
			if err != nil {
				return fmt.Errorf("end msg: %w", err)
			}
		case stageMainTrackWaitForApprovement:
			logs.Debugf("[d:%s] default -> stageMainTrackWaitForApprovement\n", ecode)

			if checkMaintenanceMode(opts, session.Label, &session.Captcha, Message.Chat.ID, ecode, false) {
				return nil
			}

			_, err := SendProtectedMessage(opts.bot, Message.Chat.ID, Message.MessageID, false, MainTrackWarnWaitForApprovement, ecode)
			if err != nil {
				return fmt.Errorf("wait msg: %w", err)
			}
		case stageMainTrackStart, stageMainTrackWaitForWanting:
			logs.Debugf("[d:%s] default -> stageMainTrackStart/stageMainTrackWaitForWanting\n", ecode)

			label := onlyBase64Symbols.ReplaceAllString(Message.CommandArguments(), "")
			if len(label) > 64 {
				label = label[:64]
			}

			sessionLabel := SessionLabel{
				Label: label,
				Time:  time.Now(),
				ID:    uuid.New(),
			}

			if session.Label.Label != "" &&
				(!session.Label.Time.IsZero()) &&
				session.Label.ID != uuid.Nil {
				sessionLabel = session.Label
			}

			if command == "start" && session.Stage == stageMainTrackStart {
				x := rand.Intn(len(MainTrackQuizMessage))
				for prefix := range MainTrackQuizMessage {
					if x == 0 {
						label = prefix + label
						if len(label) > 64 {
							label = label[:64]
						}

						sessionLabel.Label = label

						break
					}

					x--
				}

				if err := opts.ls.Update(sessionLabel); err != nil {
					return fmt.Errorf("update label: %w", err)
				}
			}

			if session.Stage != stageMainTrackWaitForWanting && !warnAutodeleteSettings(opts, Message.Chat.ID, ecode) {
				fmt.Fprintf(os.Stderr, "new session: %#v\n", session)

				session.Label = setLabel(session.Label, MarkerEmptyLabel)

				setSession(opts.db, opts.sessionSecret, sessionLabel, &session.Captcha, Message.Chat.ID, 0, 0, stageMainTrackStart, SessionStatePayloadBan, nil)

				return nil
			}

			if !checkCaptcha(opts, &session.Captcha, session.Label, Message.Chat.ID, ecode, session.Stage, session.State, session.Payload) {
				return nil
			}

			if err := sendWelcomeMessage(opts, sessionLabel, &session.Captcha, Message.Chat.ID, lang); err != nil {
				return fmt.Errorf("welcome msg: %w", err)
			}
		case stageMainTrackWaitForBill:
			logs.Debugf("[d:%s] default -> stageMainTrackWaitForBill\n", ecode)

			if checkMaintenanceMode(opts, session.Label, &session.Captcha, Message.Chat.ID, ecode, false) {
				return nil
			}

			if err := checkBillMessageMessage(opts, session.Label, &session.Captcha, Message, ecode, lang); err != nil {
				return fmt.Errorf("bill recv: %w", err)
			}
		default:
			logs.Debugf("[d:%s] default -> unknown stage: %d\n", ecode, session.Stage)

			if checkMaintenanceMode(opts, session.Label, &session.Captcha, Message.Chat.ID, ecode, false) {
				return nil
			}

			if _, err := SendProtectedMessage(opts.bot, Message.Chat.ID, 0, false, InfoUnknownCommandMessage, ecode); err != nil {
				return fmt.Errorf("unknown cmd: %w", err)
			}
		}
	}

	return nil
}

// SendOpenMessage - send common message.
func SendOpenMessage(bot *tgbotapi.BotAPI, chatID int64, replyID int, pv bool, text, ecode string) (*tgbotapi.Message, error) {
	msg, err := sendMessage(bot, chatID, replyID, false, pv, text, ecode)
	if err != nil {
		return msg, fmt.Errorf("open msg: %w", err)
	}

	return msg, nil
}

// SendProtectedMessage - send message with protected content.
func SendProtectedMessage(bot *tgbotapi.BotAPI, chatID int64, replyID int, pv bool, text, ecode string) (*tgbotapi.Message, error) {
	msg, err := sendMessage(bot, chatID, replyID, true, pv, text, ecode)
	if err != nil {
		return msg, fmt.Errorf("protect msg: %w", err)
	}

	return msg, nil
}

// SendMessage - send common message.
func sendMessage(bot *tgbotapi.BotAPI, chatID int64, replyID int, protect, preview bool, text, ecode string) (*tgbotapi.Message, error) {
	logs.Debugf("[!:%s] send message\n", ecode)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ProtectContent = protect
	msg.DisableWebPagePreview = !preview

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

// checkMaintenanceMode - check maintenance mode.
func checkMaintenanceMode(opts handlerOpts, label SessionLabel, c *SessionCaptcha, chatID int64, ecode string, whenfull bool) bool {
	mntFull, mntNewreg := opts.mnt.Check()

	fmt.Fprintf(os.Stderr, "mntFull: %v mntNewreg: %v, whenfull: %v\n", mntFull != "", mntNewreg != "", whenfull)

	if mntFull != "" || (mntNewreg != "" && !whenfull) {
		text := mntNewreg
		if mntFull != "" {
			text = mntFull
		}

		_, err := SendProtectedMessage(opts.bot, chatID, 0, false, text, ecode)
		if err != nil {
			if IsForbiddenError(err) {
				setSession(opts.db, opts.sessionSecret, label, c, chatID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)
			}
		}

		return true
	}

	return false
}
