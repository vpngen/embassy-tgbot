package main

import (
	"fmt"
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
		SendProtectMessage(opts.bot, update.Message.Chat.ID, 0, WarnForbidForwards, ecode)

		return
	}
}

// Handling callbacks  (opposed messages).
func buttonHandler2(opts hOpts, update tgbotapi.Update) {
	var key []byte

	defer opts.wg.Done()

	ecode := genEcode() // unique error code

	switch {
	case strings.HasPrefix(update.CallbackQuery.Data, acceptReceiptPrefix):
		id := strings.TrimPrefix(update.CallbackQuery.Data, acceptReceiptPrefix)

		logs.Debugf("[!:%s]accept receipt: %s\n", ecode, id)
		fmt.Sscanf(id, "%x", &key)

		err := UpdateReceipt2(opts.db, key, CkReceiptStageDecision2, true)
		if err != nil {
			logs.Errf("[!:%s] set receipt: %s\n", ecode, err)

			return
		}

		//ResetReceipt2(opts.db, key)

		if len(update.CallbackQuery.Message.Photo) > 0 {
			photo := tgbotapi.NewPhoto(update.CallbackQuery.Message.Chat.ID, tgbotapi.FileID(update.CallbackQuery.Message.Photo[0].FileID))
			// msg.ReplyMarkup = WannabeKeyboard
			photo.ParseMode = tgbotapi.ModeMarkdown
			photo.ProtectContent = true
			text := "\U00002705" + ` *Accept receipt*
Message date: *%s*
Action date: *%s*
Admin: [%s](tg://user?id=%d) (@%s)
			`
			cbq := update.CallbackQuery
			photo.Caption = fmt.Sprintf(
				text,
				cbq.Message.Time().Format(time.RFC3339),
				time.Now().Format(time.RFC3339),
				cbq.From.FirstName+" "+cbq.From.LastName, cbq.From.ID, cbq.From.UserName,
			)

			if _, err := opts.bot.Request(photo); err != nil {
				logs.Errf("[!:%s] request2: %s", ecode, err)
			}
		}

		// delete our previous message.
		if err := RemoveMsg(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
			// we don't want to handle this
			logs.Errf("[!:%s] remove: %s\n", ecode, err)
		}

	case strings.HasPrefix(update.CallbackQuery.Data, rejectReceiptPrefix):
		id := strings.TrimPrefix(update.CallbackQuery.Data, rejectReceiptPrefix)

		logs.Debugf("[!:%s]reject receipt: %s\n", ecode, id)
		fmt.Sscanf(id, "%x", &key)

		err := UpdateReceipt2(opts.db, key, CkReceiptStageDecision2, false)
		if err != nil {
			logs.Errf("[!:%s] set receipt: %s\n", ecode, err)

			return
		}

		//ResetReceipt2(opts.db, key)

		if len(update.CallbackQuery.Message.Photo) > 0 {
			photo := tgbotapi.NewPhoto(update.CallbackQuery.Message.Chat.ID, tgbotapi.FileID(update.CallbackQuery.Message.Photo[0].FileID))
			// msg.ReplyMarkup = WannabeKeyboard
			photo.ParseMode = tgbotapi.ModeMarkdown
			photo.ProtectContent = true
			text := "\U0000274C" + ` *Reject receipt*
Message date: *%s*
Action date: *%s*
Admin: [%s](tg://user?id=%d) (@%s)
			`
			cbq := update.CallbackQuery
			photo.Caption = fmt.Sprintf(
				text,
				cbq.Message.Time().Format(time.RFC3339),
				time.Now().Format(time.RFC3339),
				cbq.From.FirstName+" "+cbq.From.LastName, cbq.From.ID, cbq.From.UserName,
			)

			if _, err := opts.bot.Request(photo); err != nil {
				logs.Errf("[!:%s] request2: %s", ecode, err)
			}
		}

		// delete our previous message.
		if err := RemoveMsg(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
			// we don't want to handle this
			logs.Errf("[!:%s] remove: %s\n", ecode, err)
		}

	default:
	}
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
