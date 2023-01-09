package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vpngen/embassy-tgbot/logs"
)

// Handling messages (opposed callback).
func messageHandler2(opts hOpts, update tgbotapi.Update) {
	defer opts.wg.Done()

	ecode := genEcode() // unique e-code

	// delete our previous message.
	defer func() {
		if err := RemoveMsg(opts.bot, update.Message.Chat.ID, update.Message.MessageID); err != nil {
			// we don't want to handle this
			logs.Errf("[!:%s] remove: %s\n", ecode, err)
		}
	}()

	if update.Message.ForwardFrom != nil ||
		update.Message.ForwardFromChat != nil {
		SendProtectedMessage(opts.bot, update.Message.Chat.ID, 0, InfoForbidForwardsMessage, ecode)

		return
	}
}

// Handling callbacks  (opposed messages).
func buttonHandler2(opts hOpts, update tgbotapi.Update) {
	var key []byte

	defer opts.wg.Done()

	ecode := genEcode() // unique error code

	decisionPrefix, id, ok := strings.Cut(update.CallbackQuery.Data, "-")
	if !ok {
		return
	}

	switch {
	case strings.HasPrefix(decisionPrefix, acceptReceiptPrefix):
		reasonString := strings.TrimPrefix(decisionPrefix, acceptReceiptPrefix)

		reason, err := strconv.Atoi(reasonString)
		if err != nil {
			logs.Errf("[!:%s] reason atoi: %s: %s\n", ecode, err, reasonString)

			return
		}

		logs.Debugf("[!:%s]accept receipt: %s: %s: %d\n", ecode, id, reasonString, reason)
		fmt.Sscanf(id, "%x", &key)

		err = UpdateReceipt2(opts.db, key, CkReceiptStageDecision2, true, reason)
		if err != nil {
			logs.Errf("[!:%s] set receipt: %s\n", ecode, err)

			return
		}

		//ResetReceipt2(opts.db, key)

		if err := saveDecision(opts.bot, *update.CallbackQuery, reason, decisionAdminAccept, ecode); err != nil {
			logs.Errf("[!:%s] repost photo: %s\n", ecode, err)
		}

		// delete our previous message.
		if err := RemoveMsg(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
			// we don't want to handle this
			logs.Errf("[!:%s] remove: %s\n", ecode, err)
		}

	case strings.HasPrefix(decisionPrefix, rejectReceiptPrefix):
		reasonString := strings.TrimPrefix(decisionPrefix, rejectReceiptPrefix)

		reason, err := strconv.Atoi(reasonString)
		if err != nil {
			logs.Errf("[!:%s] reason atoi: %s: %s\n", ecode, err, reasonString)

			return
		}

		logs.Debugf("[!:%s]reject receipt: %s: %s: %s\n", ecode, id, reasonString, reason)
		fmt.Sscanf(id, "%x", &key)

		err = UpdateReceipt2(opts.db, key, CkReceiptStageDecision2, false, reason)
		if err != nil {
			logs.Errf("[!:%s] set receipt: %s\n", ecode, err)

			return
		}

		//ResetReceipt2(opts.db, key)

		if err := saveDecision(opts.bot, *update.CallbackQuery, reason, decisionAdminReject, ecode); err != nil {
			logs.Errf("[!:%s] repost photo: %s\n", ecode, err)
		}

		// delete our previous message.
		if err := RemoveMsg(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
			// we don't want to handle this
			logs.Errf("[!:%s] remove: %s\n", ecode, err)
		}

	default:
	}
}

var (
	decisionAdminComment = `
Action: %s
Message date: *%s*
Action date: *%s*
Admin: [%s](tg://user?id=%d)`

	decisionAdminReject = "\U0000274C" + ` *Reject receipt*` + decisionAdminComment

	decisionAdminAccept = "\U00002705" + ` *Accept receipt*` + decisionAdminComment
)

func saveDecision(bot *tgbotapi.BotAPI, cbq tgbotapi.CallbackQuery, reason int, comment, ecode string) error {
	if len(cbq.Message.Photo) > 0 {
		photo := tgbotapi.NewPhoto(cbq.Message.Chat.ID, tgbotapi.FileID(cbq.Message.Photo[0].FileID))
		// msg.ReplyMarkup = WannabeKeyboard
		photo.ParseMode = tgbotapi.ModeMarkdown
		photo.ProtectContent = true
		photo.Caption = fmt.Sprintf(
			comment,
			tgbotapi.EscapeText(tgbotapi.ModeMarkdown, buttons[reason]),
			tgbotapi.EscapeText(tgbotapi.ModeMarkdown, cbq.Message.Time().Format(time.RFC3339)),
			tgbotapi.EscapeText(tgbotapi.ModeMarkdown, time.Now().Format(time.RFC3339)),
			tgbotapi.EscapeText(tgbotapi.ModeMarkdown, cbq.From.FirstName+" "+cbq.From.LastName),
			cbq.From.ID,
		)

		if cbq.From.UserName != "" {
			photo.Caption += " (@" + tgbotapi.EscapeText(tgbotapi.ModeMarkdown, cbq.From.UserName) + ")"

		}

		if _, err := bot.Request(photo); err != nil {
			return fmt.Errorf("request: %w", err)
		}
	}

	return nil
}

func handleCommands2(opts hOpts, Message *tgbotapi.Message, session *Session, ecode string) error {
	/*switch Message.Command() {
	case "reset":
		if opts.debug == int(logs.LevelDebug) {
			err := resetSession(opts.db, Message.Chat.ID)
			if err == nil {
				SendMessage(opts.bot, Message.Chat.ID, 0, ResetSuccessfull, ecode)
			}

			return err
		}

		fallthrough
	default:
		SendMessage(opts.bot, Message.Chat.ID, 0, WarnUnknownCommand, ecode)

		return nil
	}*/

	return nil
}
