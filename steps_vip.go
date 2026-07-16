package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/vpngen/embassy-tgbot/logs"
)

// Session stages persisted in DB while the user moves through the VIP flow.
// 100+ is chosen to avoid collision with the sequential stage constants in talks.go (0–7).
const (
	stagePayTribute       = 100
	stageVIPQuiz          = 101
	stageVIPBrigadeExists = 102
	stageVIPKeydesk       = 105
)

func init() {
	RegisterStep(&Step{
		ID:         "pay_tribute",
		FlowStage:  "vip_get_urls",
		OnCallback: stepPayTributeOnCallback,
	})

	RegisterStep(&Step{
		ID:         "vip_quiz",
		FlowStage:  "vip_quiz",
		OnEnter:    stepVIPQuizOnEnter,
		OnCallback: stepVIPQuizOnCallback,
	})

	RegisterStep(&Step{
		ID:         "vip_brigade_exists",
		FlowStage:  "vip_brigade_exists",
		OnEnter:    stepVIPBrigadeExistsOnEnter,
		OnCallback: stepVIPBrigadeExistsOnCallback,
	})

	RegisterStep(&Step{
		ID:         "vip_keydesk",
		FlowStage:  "vip_keydesk",
		OnEnter:    stepVIPKeydeskOnEnter,
		OnCallback: stepVIPKeydeskOnCallback,
	})

	LegacyStageToStepID[stagePayTribute] = "pay_tribute"
	StepIDToLegacyStage["pay_tribute"] = stagePayTribute

	LegacyStageToStepID[stageVIPQuiz] = "vip_quiz"
	StepIDToLegacyStage["vip_quiz"] = stageVIPQuiz

	LegacyStageToStepID[stageVIPBrigadeExists] = "vip_brigade_exists"
	StepIDToLegacyStage["vip_brigade_exists"] = stageVIPBrigadeExists

	LegacyStageToStepID[stageVIPKeydesk] = "vip_keydesk"
	StepIDToLegacyStage["vip_keydesk"] = stageVIPKeydesk
}

// --- vip_quiz: "do you already have a brigade?" ---

// stepVIPQuizOnEnter renders vip_quiz from the VIP flow JSON (not the main
// flow), since the default Transition render path only knows about main.json.
func stepVIPQuizOnEnter(ctx *StepContext) error {
	text := ctx.FlowVipMessage("vip_quiz", "")
	kb := ctx.FlowVipKeyboard("vip_quiz", nil)

	newMsg, err := ctx.Send(text, &kb)
	if err != nil {
		return fmt.Errorf("send vip_quiz: %w", err)
	}

	return ctx.SaveSession(newMsg, stageVIPQuiz, SessionStatePayloadSomething, nil)
}

func stepVIPQuizOnCallback(ctx *StepContext, data string) error {
	switch data {
	case "vip_brigade_exists":
		if err := ctx.Transition("vip_brigade_exists", ctx.Session.State, nil); err != nil {
			return err
		}
		defer ctx.RemoveMessage(ctx.Session.OurMsgID)
		return nil

	case "vip_get_urls":
		return sendBuyVIPMessage(ctx, uuid.Nil)

	case "reset":
		return stepResetCallback(ctx)

	default:
		logs.Debugf("[!:%s] unknown callback %q in vip_quiz\n", ctx.Ecode, data)
		return nil
	}
}

// --- vip_brigade_exists: choose how to identify the existing brigade ---

// stepVIPBrigadeExistsOnEnter renders vip_brigade_exists from the VIP flow JSON.
func stepVIPBrigadeExistsOnEnter(ctx *StepContext) error {
	text := ctx.FlowVipMessage("vip_brigade_exists", "")
	kb := ctx.FlowVipKeyboard("vip_brigade_exists", nil)

	newMsg, err := ctx.Send(text, &kb)
	if err != nil {
		return fmt.Errorf("send vip_brigade_exists: %w", err)
	}

	return ctx.SaveSession(newMsg, stageVIPBrigadeExists, SessionStatePayloadSomething, nil)
}

func stepVIPBrigadeExistsOnCallback(ctx *StepContext, data string) error {
	switch data {
	case "vip_name":
		if checkVIPIdentifyLockout(ctx) {
			return nil
		}
		if err := ctx.Transition("vip_name", ctx.Session.State, nil); err != nil {
			return err
		}
		defer ctx.RemoveMessage(ctx.Session.OurMsgID)
		return nil

	case "vip_keydesk":
		if err := ctx.Transition("vip_keydesk", ctx.Session.State, nil); err != nil {
			return err
		}
		defer ctx.RemoveMessage(ctx.Session.OurMsgID)
		return nil

	case "reset":
		return stepResetCallback(ctx)

	default:
		logs.Debugf("[!:%s] unknown callback %q in vip_brigade_exists\n", ctx.Ecode, data)
		return nil
	}
}

// --- vip_keydesk: identify via the self-service key manager instead ---

// stepVIPKeydeskOnEnter renders vip_keydesk from the VIP flow JSON.
func stepVIPKeydeskOnEnter(ctx *StepContext) error {
	text := ctx.FlowVipMessage("vip_keydesk", "")
	kb := ctx.FlowVipKeyboard("vip_keydesk", nil)

	newMsg, err := ctx.Send(text, &kb)
	if err != nil {
		return fmt.Errorf("send vip_keydesk: %w", err)
	}

	return ctx.SaveSession(newMsg, stageVIPKeydesk, SessionStatePayloadSomething, nil)
}

func stepVIPKeydeskOnCallback(ctx *StepContext, data string) error {
	switch data {
	case "reset":
		return stepResetCallback(ctx)

	default:
		logs.Debugf("[!:%s] unknown callback %q in vip_keydesk\n", ctx.Ecode, data)
		return nil
	}
}

func stepPayTributeOnCallback(ctx *StepContext, data string) error {
	switch data {
	case "vip_month":
		return handleVIPPlanSelection(ctx, "vip_pay_month")
	case "vip_year":
		return handleVIPPlanSelection(ctx, "vip_pay_year")
	case "reset":
		return stepResetCallback(ctx)
	default:
		return nil
	}
}

// handleVIPPlanSelection sends the tribute payment link for the chosen plan.
// Brigade reservation already happened in sendBuyVIPMessage.
func handleVIPPlanSelection(ctx *StepContext, payStageID string) error {
	text := ctx.FlowVipMessage(payStageID, "")
	kb := ctx.FlowVipKeyboard(payStageID, nil)
	_, err := ctx.Send(text, &kb)
	return err
}

// sendBuyVIPMessage reserves a brigade, links it with the
// partner API, then shows the month/year plan selection buttons.
// brigadeID is optional (uuid.Nil means "reserve a new brigade"); when set,
// it identifies an existing brigade (e.g. resolved via name+6-words lookup)
// to upgrade to VIP instead.
func sendBuyVIPMessage(ctx *StepContext, brigadeID uuid.UUID) error {
	// Ensure the session has a label (same logic as stepVIPCallback).
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

	if ctx.Session.Label.Time.IsZero() || ctx.Session.Label.ID == uuid.Nil {
		ctx.Session.Label = SessionLabel{
			Label: sessionLabel,
			Time:  time.Now(),
			ID:    uuid.New(),
		}
	}

	if ctx.Session.Label.Label == "" {
		ctx.Session.Label.Label = sessionLabel
	}

	bid := ""
	if brigadeID != uuid.Nil {
		bid = brigadeID.String()
	}

	// Reserve a brigade in ministry.
	brigadeUUID, err := reqBrigade(ctx.Dept, ctx.ChatID, ctx.Session.Label, bid, ctx.Lang)
	if err != nil || brigadeUUID == uuid.Nil {
		ctx.Wrong(fmt.Errorf("reserve brigade failed"))
		return nil
	}

	if err := reserveVIPWithMinistry(ctx.Dept, brigadeUUID, ctx.ChatID); err != nil {
		return fmt.Errorf("ministry reserve: %w", err)
	}

	// Send the plan selection message with month/year buttons.
	text := ctx.FlowVipMessage("vip_get_urls", "")
	kb := ctx.FlowVipKeyboard("vip_get_urls", nil)

	newMsg, err := ctx.Send(text, &kb)
	if err != nil {
		return fmt.Errorf("send pay tribute msg: %w", err)
	}

	defer ctx.RemoveMessage(ctx.Session.OurMsgID)

	return ctx.SaveSession(newMsg, stagePayTribute, ctx.Session.State, nil)
}
