package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// stagePayTribute is the session stage persisted in DB while the user is choosing a VIP plan.
// 100 is chosen to avoid collision with the sequential stage constants in talks.go (0–7).
const stagePayTribute = 100

func init() {
	RegisterStep(&Step{
		ID:         "pay_tribute",
		FlowStage:  "vip_get_urls",
		OnCallback: stepPayTributeOnCallback,
	})

	LegacyStageToStepID[stagePayTribute] = "pay_tribute"
	StepIDToLegacyStage["pay_tribute"] = stagePayTribute
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
func sendBuyVIPMessage(ctx *StepContext) error {
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

	// Reserve a brigade in ministry.
	brigadeUUID, err := reqBrigade(ctx.Dept, ctx.ChatID, ctx.Session.Label, "", ctx.Lang)
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
