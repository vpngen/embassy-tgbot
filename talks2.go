package main

import (
	"strings"

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
		SendMessage(opts.bot, update.Message.Chat.ID, 0, WarnForbidForwards, ecode)

		return
	}
}

// Handling callbacks  (opposed messages).
func buttonHandler2(opts hOpts, update tgbotapi.Update) {
	defer opts.wg.Done()

	ecode := genEcode() // unique error code

	switch {
	case strings.HasPrefix(update.CallbackQuery.Data, acceptPrefix):
		id := strings.TrimPrefix(update.CallbackQuery.Data, acceptPrefix)
		logs.Debugf("accept bill: %s\n", id)

		// delete our previous message.
		if err := RemoveMsg(opts.bot, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID); err != nil {
			// we don't want to handle this
			logs.Errf("[!:%s] remove: %s\n", ecode, err)
		}

	case strings.HasPrefix(update.CallbackQuery.Data, rejectPrefix):
		id := strings.TrimPrefix(update.CallbackQuery.Data, rejectPrefix)
		logs.Debugf("reject bill: %s\n", id)

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
