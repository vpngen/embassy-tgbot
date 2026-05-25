package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	tgbotapi "github.com/vpngen/embassy-tgbot/telegram-bot-api"

	"github.com/vpngen/embassy-tgbot/logs"
)

func init() {
	RegisterStep(&Step{
		ID:               "main_start",
		FlowStage:        "welcome",
		FallbackMessage:  MainTrackWelcomeMessage,
		FallbackKeyboard: &WannabeKeyboard,
		OnMessage:        stepMainStartOnMessage,
		OnCallback:       stepMainStartOnCallback,
	})

	RegisterStep(&Step{
		ID:               "welcome",
		FlowStage:        "welcome",
		FallbackMessage:  MainTrackWelcomeMessage,
		FallbackKeyboard: &WannabeKeyboard,
		OnMessage:        stepWelcomeOnMessage,
		OnCallback:       stepWelcomeOnCallback,
	})

	RegisterStep(&Step{
		ID:              "wait_for_bill",
		FlowStage:       "quiz",
		FallbackMessage: "", // dynamic: selected from MainTrackQuizMessage
		Input:           InputPhoto,
		OnEnter:         stepQuizEnter,
		OnMessage:       stepBillOnMessage,
	})

	RegisterStep(&Step{
		ID:              "wait_for_approval",
		FlowStage:       "send_for_attestation",
		FallbackMessage: MainTrackSendForAttestationMessage,
		Input:           InputNone,
		OnMessage:       stepApprovalOnMessage,
	})

	RegisterStep(&Step{
		ID:              "cleanup",
		FlowStage:       "conversation_finished",
		FallbackMessage: MainTrackWarnConversationsFinished,
		Input:           InputNone,
		OnMessage:       stepCleanupOnMessage,
		OnCallback:      stepCleanupOnCallback,
	})

	RegisterStep(&Step{
		ID:      "faq_item",
		Input:   InputNone,
		OnEnter: stepFAQItemEnter,
	})
}

// --- main_start ---

func stepMainStartOnMessage(ctx *StepContext, msg *tgbotapi.Message) error {
	// User lands here on very first interaction or after reset.
	// Same as welcome: show the welcome message.
	if warnAutodeleteSettings(ctx.Opts, ctx.ChatID, ctx.Ecode) {
		ctx.Session.Label = setLabel(ctx.Session.Label, MarkerEmptyLabel)
		if !checkCaptcha(ctx.Opts, &ctx.Session.Captcha, ctx.Session.Label, ctx.ChatID, ctx.Ecode, ctx.Session.Stage, ctx.Session.State, ctx.Session.Payload) {
			return nil
		}
		return ctx.Transition("welcome", SessionStatePayloadSomething, nil)
	}
	return nil
}

func stepMainStartOnCallback(ctx *StepContext, data string) error {
	return stepWelcomeOnCallback(ctx, data)
}

// --- welcome ---

func stepWelcomeOnMessage(ctx *StepContext, msg *tgbotapi.Message) error {
	// User sent a text message while in welcome stage — re-show welcome.
	return ctx.Transition("welcome", SessionStatePayloadSomething, nil)
}

func stepWelcomeOnCallback(ctx *StepContext, data string) error {
	switch data {
	case "started":
		// "I want a brigade" button.
		if ctx.CheckMaintenance(false) {
			return nil
		}
		if err := stepQuizEnter(ctx); err != nil {
			return err
		}
		// Delete the welcome message.
		defer ctx.RemoveMessage(ctx.Session.OurMsgID)
		return nil

	case "continue":
		// "Continue" button.
		if ctx.CheckMaintenance(false) {
			return nil
		}
		if !checkCaptcha(ctx.Opts, &ctx.Session.Captcha, ctx.Session.Label, ctx.ChatID, ctx.Ecode, ctx.Session.Stage, ctx.Session.State, ctx.Session.Payload) {
			return nil
		}
		return ctx.Transition("welcome", SessionStatePayloadSomething, nil)

	case "vip":
		return stepVIPCallback(ctx)

	case "vip_get_urls":
		if ctx.Opts.betaChatIDs != nil && !ctx.Opts.betaChatIDs[ctx.ChatID] {
			return stepVIPCallback(ctx)
		}
		return sendBuyVIPMessage(ctx)

	case "restore":
		if ctx.CheckMaintenance(true) {
			return nil
		}
		if err := ctx.Transition("restore_start", 0, nil); err != nil {
			return err
		}
		defer ctx.RemoveMessage(ctx.Session.OurMsgID)
		return nil

	case "reset":
		return stepResetCallback(ctx)

	case "outline_download_urls":
		return sendDownloadOutlineMessage(ctx.Opts.bot, ctx.ChatID, ctx.Lang, ctx.Opts.flowMainUrl)

	case "amnezia_vpn_download_urls":
		return sendDownloadAmneziaVPNMessage(ctx.Opts.bot, ctx.ChatID, ctx.Opts.flowMainUrl)

	default:
		logs.Debugf("[!:%s] unknown callback %q in welcome step\n", ctx.Ecode, data)
		return nil
	}
}

// stepFAQItemEnter renders whichever flow stage is stored in the session payload.
func stepFAQItemEnter(ctx *StepContext) error {
	stageID := string(ctx.Session.Payload)
	if stageID == "" {
		stageID = "FAQ"
	}
	text := ctx.FlowMessage(stageID, "")
	kb := ctx.FlowKeyboard(stageID, nil)
	newMsg, err := ctx.Send(text, &kb)
	if err != nil {
		return err
	}
	return ctx.SaveSession(newMsg, stageFAQTrack, SessionStatePayloadSomething, ctx.Session.Payload)
}

func stepQuizEnter(ctx *StepContext) error {
	num, _, _ := strings.Cut(ctx.Session.Label.Label, "_")

	text := MainTrackQuizMessage[num+"_"]
	if text == "" {
		for prefix := range MainTrackQuizMessage {
			text = MainTrackQuizMessage[prefix]
			break
		}
	}

	text = ctx.FlowMessage("quiz", text)

	newMsg, err := SendProtectedMessage(ctx.Opts.bot, ctx.ChatID, 0, false, text, ctx.Ecode)
	if err != nil {
		return fmt.Errorf("send quiz: %w", err)
	}

	return ctx.SaveSession(newMsg, stageMainTrackWaitForBill, SessionStatePayloadSomething, nil)
}

func stepBillOnMessage(ctx *StepContext, msg *tgbotapi.Message) error {
	if ctx.CheckMaintenance(false) {
		return nil
	}

	if len(msg.Photo) == 0 {
		ctx.SendReply(msg.MessageID, ctx.FlowMessage("warn_required_photo", MainTrackWarnRequiredPhoto))
		return nil
	}

	_, _, count, err := catchFirstReceipt(ctx.Opts.db, CkReceiptStageNone)
	if err != nil {
		return fmt.Errorf("catch: %w", err)
	}

	if count > 10 {
		SendProtectedMessage(ctx.Opts.bot, ctx.ChatID, msg.MessageID, false,
			ctx.FlowMessage("too_busy", MainTrackWeAreSoBusy), ctx.Ecode)
		return nil
	}

	// Pick highest-resolution photo.
	photoIDX := 0
	w := 0
	for i := range msg.Photo {
		if msg.Photo[i].Width > w {
			photoIDX = i
			w = msg.Photo[i].Width
		}
	}

	logs.Debugf("photo ID: %s\n", msg.Photo[photoIDX].FileID)

	if err := PutReceipt(ctx.Opts.db, ctx.Opts.queueSecret, ctx.ChatID, msg.Photo[photoIDX].FileID, ctx.Lang); err != nil {
		return fmt.Errorf("put: %w", err)
	}

	newMsg, err := SendProtectedMessage(ctx.Opts.bot, ctx.ChatID, msg.MessageID, false,
		ctx.FlowMessage("send_for_attestation", MainTrackSendForAttestationMessage), ctx.Ecode)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return ctx.SaveSession(newMsg, stageMainTrackWaitForApprovement, SessionStatePayloadSomething, nil)
}

// --- wait_for_approval ---

func stepApprovalOnMessage(ctx *StepContext, msg *tgbotapi.Message) error {
	if ctx.CheckMaintenance(false) {
		return nil
	}
	ctx.SendReply(msg.MessageID, ctx.FlowMessage("warn_wait_approval", MainTrackWarnWaitForApprovement))
	return nil
}

// --- cleanup ---

func stepCleanupOnMessage(ctx *StepContext, msg *tgbotapi.Message) error {
	ctx.SendReply(msg.MessageID, ctx.FlowMessage("conversation_finished", MainTrackWarnConversationsFinished))
	return nil
}

func stepCleanupOnCallback(ctx *StepContext, data string) error {
	switch data {
	case "restore":
		if ctx.CheckMaintenance(true) {
			return nil
		}

		prev := ctx.Session.State
		if prev == SessionStatePayloadBan {
			ctx.SendReply(ctx.Session.OurMsgID, ctx.FlowMessage("conversation_finished", MainTrackWarnConversationsFinished))
			return nil
		}

		if err := ctx.Transition("restore_start", prev, nil); err != nil {
			return err
		}
		defer ctx.RemoveMessage(ctx.Session.OurMsgID)
		return nil

	case "reset":
		return stepResetCallback(ctx)

	default:
		return nil
	}
}

// --- Shared callback handlers ---

func stepResetCallback(ctx *StepContext) error {
	if ctx.Session.State == SessionStatePayloadSecondary {
		ctx.SendReply(ctx.Session.OurMsgID, ctx.FlowMessage("conversation_finished", MainTrackWarnConversationsFinished))
	}

	ctx.Session.Label = setLabel(ctx.Session.Label, MarkerResetLabel)

	if err := ctx.Transition("welcome", SessionStatePayloadSomething, nil); err != nil {
		return err
	}
	defer ctx.RemoveMessage(ctx.Session.OurMsgID)
	return nil
}

func stepVIPCallback(ctx *StepContext) error {
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

	requestID, err := reqBrigade(ctx.Dept, ctx.ChatID, ctx.Session.Label, "", ctx.Lang)
	if err != nil || requestID == uuid.Nil {
		ctx.Wrong(fmt.Errorf("request brigade failed"))
		return nil
	}

	vipURL := VIPBotURL + "?start=" + requestID.String()
	text := ctx.FlowVipMessage("vip_redirect", VIPMessage)
	kb := ctx.FlowVipKeyboard("vip_redirect", nil, map[string]string{"vip_bot_url": vipURL})

	// Fallback keyboard if flow doesn't have one.
	if len(kb.InlineKeyboard) == 0 {
		kb = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("Перейти в VIP-бот", vipURL),
				tgbotapi.NewInlineKeyboardButtonData("Передумал", "reset"),
			),
		)
	}

	newMsg, err := ctx.Send(text, &kb)
	if err != nil {
		return fmt.Errorf("vip send: %w", err)
	}

	if err := ctx.SaveSession(newMsg, stageMainTrackStart, SessionStatePayloadSomething, nil); err != nil {
		return err
	}

	defer ctx.RemoveMessage(ctx.Session.OurMsgID)
	return nil
}

// --- Helpers used by restore steps too ---

func parseRestoreName(text string) (string, bool) {
	cleaned := strings.Join(
		strings.Fields(
			strings.TrimSpace(
				strings.Replace(
					strings.Replace(text, ",", " ", -1),
					"\"", "", -1),
			),
		),
		" ",
	)

	_, _, ok := strings.Cut(cleaned, " ")
	if !ok || !utf8.ValidString(cleaned) {
		return "", false
	}

	return cleaned, true
}
