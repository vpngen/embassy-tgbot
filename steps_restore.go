package main

import (
	"fmt"
	"strings"

	tgbotapi "github.com/vpngen/embassy-tgbot/telegram-bot-api"

	"github.com/vpngen/embassy-tgbot/logs"
)

func init() {
	RegisterStep(&Step{
		ID:               "restore_start",
		FlowStage:        "restore_start",
		FallbackMessage:  RestoreTrackStartMessage,
		FallbackKeyboard: &RestoreStartKeyboard,
		OnMessage:        stepRestoreStartOnMessage,
		OnCallback:       stepRestoreStartOnCallback,
	})

	RegisterStep(&Step{
		ID:               "restore_name",
		FlowStage:        "restore_name",
		FallbackMessage:  RestoreTrackNameMessage,
		FallbackKeyboard: &RestoreNameKeyboard,
		Input:            InputText,
		OnMessage:        stepRestoreNameOnMessage,
		OnCallback:       stepRestoreNameOnCallback,
	})

	RegisterStep(&Step{
		ID:               "restore_words",
		FlowStage:        "restore_words",
		FallbackMessage:  RestoreTrackWordsMessage,
		FallbackKeyboard: &RestoreWordsKeyboard1,
		Input:            InputText,
		OnMessage:        stepRestoreWordsOnMessage,
		OnCallback:       stepRestoreWordsOnCallback,
	})

	RegisterStep(&Step{
		ID:              "restore_cleanup",
		FlowStage:       "conversation_finished",
		FallbackMessage: MainTrackWarnConversationsFinished,
		OnMessage:       stepCleanupOnMessage, // reuse from steps_main
		OnCallback:      stepCleanupOnCallback,
	})
}

// --- restore_start ---

func stepRestoreStartOnMessage(ctx *StepContext, msg *tgbotapi.Message) error {
	if ctx.CheckMaintenance(true) {
		return nil
	}
	// User sent text instead of pressing button — re-show the restore start.
	return ctx.Transition("restore_start", ctx.Session.State, nil)
}

func stepRestoreStartOnCallback(ctx *StepContext, data string) error {
	switch data {
	case "restore", "restore_name":
		if ctx.CheckMaintenance(true) {
			return nil
		}
		if err := ctx.Transition("restore_name", ctx.Session.State, nil); err != nil {
			return err
		}
		defer ctx.RemoveMessage(ctx.Session.OurMsgID)
		return nil

	case "reset":
		return stepResetCallback(ctx)

	default:
		logs.Debugf("[!:%s] unknown callback %q in restore_start\n", ctx.Ecode, data)
		return nil
	}
}

// --- restore_name ---

func stepRestoreNameOnMessage(ctx *StepContext, msg *tgbotapi.Message) error {
	if ctx.CheckMaintenance(true) {
		return nil
	}

	// Remove keyboard from previous message after processing.
	defer ctx.RemovePreviousKeyboard(ctx.FlowMessage("restore_name", RestoreTrackNameMessage))

	name, ok := parseRestoreName(msg.Text)
	if !ok {
		// Invalid name — show error and stay on this step.
		text := ctx.FlowMessage("restore_name_fail", RestoreTrackInvalidNameMessageVIP)
		kb := ctx.FlowKeyboard("restore_name_fail", &RestoreNameKeyboard)
		newMsg, err := ctx.Send(text, &kb)
		if err != nil {
			return err
		}
		return ctx.SaveSession(newMsg, stageRestoreTrackSendName, ctx.Session.State, nil)
	}

	// Valid name — move to words step, storing name as payload.
	return ctx.Transition("restore_words", ctx.Session.State, []byte(name))
}

func stepRestoreNameOnCallback(ctx *StepContext, data string) error {
	defer ctx.RemoveMessage(ctx.Session.OurMsgID)

	switch data {
	case "reset":
		return stepResetCallback(ctx)
	case "restore":
		if ctx.CheckMaintenance(true) {
			return nil
		}
		// User pressed "Continue" without sending name — re-show the same step.
		return ctx.Transition("restore_start", ctx.Session.State, nil)
	default:
		return nil
	}
}

// --- restore_words ---

func stepRestoreWordsOnMessage(ctx *StepContext, msg *tgbotapi.Message) error {
	if ctx.CheckMaintenance(true) {
		return nil
	}

	// Remove keyboard from previous message.
	defer ctx.RemovePreviousKeyboard(ctx.FlowMessage("restore_words", RestoreTrackWordsMessage))

	name := ctx.Session.Payload
	if name == nil {
		return sendWordsFailedV2(ctx, nil)
	}

	words := strings.Join(
		strings.Fields(
			strings.TrimSpace(
				strings.Replace(
					strings.Replace(msg.Text, ",", " ", -1),
					"\"", "", -1),
			),
		),
		" ",
	)

	if words == "" || len(strings.Split(words, " ")) < 6 {
		return sendWordsFailedV2(ctx, name)
	}

	err := RestoreBrigadier(ctx.Opts.bot, ctx.ChatID, ctx.Ecode, ctx.Dept, ctx.Opts.mnt, string(name), words, ctx.Lang, ctx.Opts.flowMainUrl)
	if err != nil {
		return sendWordsFailedV2(ctx, name)
	}

	// Success — move to restore_cleanup.
	return ctx.SaveSessionNoMsg(stageRestoreTrackCleanup, ctx.Session.State, nil)
}

func stepRestoreWordsOnCallback(ctx *StepContext, data string) error {
	switch data {
	case "again", "return", "restore_name":
		if ctx.CheckMaintenance(false) {
			return nil
		}

		// Remove keyboard with appropriate text.
		text := ctx.FlowMessage("restore_words", RestoreTrackWordsMessage)
		if data == "again" || data == "restore_name" {
			text = ctx.FlowMessage("restore_words_fail", RestoreTrackBrigadeNotFoundMessage)
		}
		defer func() {
			if err := RemoveKeyboardMsg(ctx.Opts.bot, ctx.ChatID, ctx.Session.OurMsgID, text); err != nil {
				logs.Errf("[!:%s] restore keyboard: %s\n", ctx.Ecode, err)
			}
		}()

		// Go back to restore_name.
		return ctx.Transition("restore_name", ctx.Session.State, nil)

	case "reset":
		return stepResetCallback(ctx)

	default:
		return nil
	}
}

// sendWordsFailedV2 shows the "brigade not found" message with retry keyboard.
func sendWordsFailedV2(ctx *StepContext, name []byte) error {
	text := ctx.FlowMessage("restore_words_fail", RestoreTrackBrigadeNotFoundMessageVIP)
	kb := ctx.FlowKeyboard("restore_words_fail", &RestoreWordsKeyboard2)

	newMsg, err := ctx.Send(text, &kb)
	if err != nil {
		return fmt.Errorf("send words failed: %w", err)
	}

	return ctx.SaveSession(newMsg, stageRestoreTrackSendWords, ctx.Session.State, name)
}
