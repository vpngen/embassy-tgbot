package main

import (
	"errors"
	"fmt"
	"strings"

	tgbotapi "github.com/vpngen/embassy-tgbot/telegram-bot-api"

	"github.com/vpngen/embassy-tgbot/logs"
)

// Session stages persisted in DB while the user identifies an existing
// brigade by name+6-words on the VIP-upgrade path. 100+ range, see steps_vip.go.
const (
	stageVIPName  = 103
	stageVIPWords = 104
)

func init() {
	RegisterStep(&Step{
		ID:         "vip_name",
		FlowStage:  "vip_name",
		Input:      InputText,
		OnEnter:    stepVIPNameOnEnter,
		OnMessage:  stepVIPNameOnMessage,
		OnCallback: stepVIPNameOnCallback,
	})

	RegisterStep(&Step{
		ID:         "vip_words",
		FlowStage:  "vip_words",
		Input:      InputText,
		OnEnter:    stepVIPWordsOnEnter,
		OnMessage:  stepVIPWordsOnMessage,
		OnCallback: stepVIPWordsOnCallback,
	})

	LegacyStageToStepID[stageVIPName] = "vip_name"
	StepIDToLegacyStage["vip_name"] = stageVIPName

	LegacyStageToStepID[stageVIPWords] = "vip_words"
	StepIDToLegacyStage["vip_words"] = stageVIPWords
}

// --- vip_name ---

func stepVIPNameOnEnter(ctx *StepContext) error {
	text := ctx.FlowVipMessage("vip_name", "")
	kb := ctx.FlowVipKeyboard("vip_name", nil)

	newMsg, err := ctx.Send(text, &kb)
	if err != nil {
		return fmt.Errorf("send vip_name: %w", err)
	}

	return ctx.SaveSession(newMsg, stageVIPName, SessionStatePayloadSomething, nil)
}

func stepVIPNameOnMessage(ctx *StepContext, msg *tgbotapi.Message) error {
	name, ok := parseRestoreName(msg.Text)
	if !ok {
		text := ctx.FlowVipMessage("vip_name_fail", "")
		kb := ctx.FlowVipKeyboard("vip_name_fail", nil)

		newMsg, err := ctx.Send(text, &kb)
		if err != nil {
			return fmt.Errorf("send vip_name_fail: %w", err)
		}

		return ctx.SaveSession(newMsg, stageVIPName, ctx.Session.State, nil)
	}

	return ctx.Transition("vip_words", ctx.Session.State, []byte(name))
}

func stepVIPNameOnCallback(ctx *StepContext, data string) error {
	switch data {
	case "reset":
		return stepResetCallback(ctx)
	default:
		logs.Debugf("[!:%s] unknown callback %q in vip_name\n", ctx.Ecode, data)
		return nil
	}
}

// --- vip_words ---

func stepVIPWordsOnEnter(ctx *StepContext) error {
	text := ctx.FlowVipMessage("vip_words", "")
	kb := ctx.FlowVipKeyboard("vip_words", nil)

	newMsg, err := ctx.Send(text, &kb)
	if err != nil {
		return fmt.Errorf("send vip_words: %w", err)
	}

	return ctx.SaveSession(newMsg, stageVIPWords, SessionStatePayloadSomething, ctx.Session.Payload)
}

func stepVIPWordsOnMessage(ctx *StepContext, msg *tgbotapi.Message) error {
	// Re-checked here (not just on the "Name + 6 words!" button press) so a
	// user can't dodge the lockout by staying on this step and resubmitting
	// words against the same name over and over.
	if checkVIPIdentifyLockout(ctx) {
		return nil
	}

	name := ctx.Session.Payload
	if name == nil {
		return sendVIPWordsFailed(ctx, nil)
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
		return sendVIPWordsFailed(ctx, name)
	}

	brigadeID, err := CheckBrigadierID(ctx.Dept, string(name), words)
	if err != nil {
		if errors.Is(err, ErrBrigadeNotFound) {
			recordVIPIdentifyFailure(ctx.Opts.db, ctx.ChatID)
		}
		return sendVIPWordsFailed(ctx, name)
	}

	return sendBuyVIPMessage(ctx, brigadeID)
}

func stepVIPWordsOnCallback(ctx *StepContext, data string) error {
	switch data {
	case "reset":
		return stepResetCallback(ctx)
	default:
		logs.Debugf("[!:%s] unknown callback %q in vip_words\n", ctx.Ecode, data)
		return nil
	}
}

// sendVIPWordsFailed shows the "couldn't identify your brigade" message and
// stays on vip_words so the next text message is retried against the same name.
func sendVIPWordsFailed(ctx *StepContext, name []byte) error {
	text := ctx.FlowVipMessage("vip_words_fail", "")
	kb := ctx.FlowVipKeyboard("vip_words_fail", nil)

	newMsg, err := ctx.Send(text, &kb)
	if err != nil {
		return fmt.Errorf("send vip_words_fail: %w", err)
	}

	return ctx.SaveSession(newMsg, stageVIPWords, ctx.Session.State, name)
}
