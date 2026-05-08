package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	tgbotapi "github.com/vpngen/embassy-tgbot/telegram-bot-api"

	"github.com/vpngen/embassy-tgbot/logs"
)

// StepID is a string identifier for a conversation step.
type StepID = string

// InputType describes what kind of user input a step expects.
type InputType int

const (
	InputNone     InputType = iota // step expects no input (button-driven)
	InputText                      // step expects a text message
	InputPhoto                     // step expects a photo
	InputReaction                  // step expects a reaction (emoji)
)

// Step defines one node in the conversation state machine.
type Step struct {
	ID        StepID
	FlowStage string // stage ID in admin-panel flow JSON (for message + keyboard)
	Input     InputType

	// Fallback message/keyboard when flow JSON is unavailable.
	FallbackMessage  string
	FallbackKeyboard *tgbotapi.InlineKeyboardMarkup

	// Handlers — only set the ones relevant to this step.
	OnMessage  func(ctx *StepContext, msg *tgbotapi.Message) error
	OnCallback func(ctx *StepContext, data string) error
	OnReaction func(ctx *StepContext, reaction tgbotapi.MessageReactionUpdated) error

	// OnEnter is called when we transition TO this step (optional).
	// If nil, the default behavior is: send FlowStage message + keyboard, save session.
	OnEnter func(ctx *StepContext) error
}

// StepContext carries all deps and per-request state for a single handler invocation.
type StepContext struct {
	Opts    handlerOpts
	Session *Session
	ChatID  int64
	Ecode   string
	Lang    string
	Dept    MinistryOpts // ministry options (only needed for some steps)
}

// --- Registry ---

var stepRegistry = map[StepID]*Step{}

// RegisterStep adds a step to the global registry.
func RegisterStep(s *Step) {
	if _, exists := stepRegistry[s.ID]; exists {
		panic(fmt.Sprintf("duplicate step registration: %q", s.ID))
	}
	stepRegistry[s.ID] = s
}

// GetStep looks up a step by ID.
func GetStep(id StepID) *Step {
	return stepRegistry[id]
}

// --- Mapping from legacy int stages to StepIDs ---

// LegacyStageToStepID maps the old integer stage constants to new step IDs.
// This allows the new dispatcher to work with sessions created by the old code.
var LegacyStageToStepID = map[int]StepID{
	stageMainTrackStart:              "main_start",
	stageMainTrackWaitForWanting:     "welcome",
	stageMainTrackWaitForBill:        "wait_for_bill",
	stageMainTrackWaitForApprovement: "wait_for_approval",
	stageMainTrackCleanup:            "cleanup",
	stageRestoreTrackStart:           "restore_start",
	stageRestoreTrackSendName:        "restore_name",
	stageRestoreTrackSendWords:       "restore_words",
	stageRestoreTrackCleanup:         "restore_cleanup",
}

// StepIDToLegacyStage is the reverse mapping (new step → old int stage).
// Needed so we can write sessions the old code can still read.
var StepIDToLegacyStage = map[StepID]int{}

func init() {
	for k, v := range LegacyStageToStepID {
		StepIDToLegacyStage[v] = k
	}
}

// --- StepContext helpers ---

// FlowMessage fetches a message from the flow JSON, falling back to the constant.
func (c *StepContext) FlowMessage(stageID, fallback string) string {
	return flowMessage(c.Opts.flowMainUrl, stageID, fallback, c.Lang)
}

// FlowVipMessage fetches a message from the vip flow JSON, falling back to the constant.
func (c *StepContext) FlowVipMessage(stageID, fallback string) string {
	return flowMessage(c.Opts.flowVipUrl, stageID, fallback, c.Lang)
}

// FlowVipKeyboard fetches a keyboard from the vip flow JSON, falling back to the provided one.
func (c *StepContext) FlowVipKeyboard(stageID string, fallback *tgbotapi.InlineKeyboardMarkup, extraVars ...map[string]string) tgbotapi.InlineKeyboardMarkup {
	if kb, ok := flowKeyboard(c.Opts.flowVipUrl, stageID, c.Opts.supportURL, c.Lang, extraVars...); ok {
		return *kb
	}
	if fallback != nil {
		return *fallback
	}
	return tgbotapi.InlineKeyboardMarkup{}
}

// FlowKeyboard fetches a keyboard from the flow JSON, falling back to the provided one.
func (c *StepContext) FlowKeyboard(stageID string, fallback *tgbotapi.InlineKeyboardMarkup, extraVars ...map[string]string) tgbotapi.InlineKeyboardMarkup {
	if kb, ok := flowKeyboard(c.Opts.flowMainUrl, stageID, c.Opts.supportURL, c.Lang, extraVars...); ok {
		return *kb
	}
	if fallback != nil {
		return *fallback
	}
	return tgbotapi.InlineKeyboardMarkup{}
}

// Send sends a protected message and returns the sent message.
// Handles IsForbiddenError internally by banning the user.
func (c *StepContext) Send(text string, kb *tgbotapi.InlineKeyboardMarkup) (*tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(c.ChatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = true
	if kb != nil {
		msg.ReplyMarkup = *kb
	}

	newMsg, err := c.Opts.bot.Send(msg)
	if err != nil {
		if IsForbiddenError(err) {
			c.Ban()
			return nil, fmt.Errorf("forbidden: %w", err)
		}
		return nil, fmt.Errorf("send: %w", err)
	}

	return &newMsg, nil
}

// SendPlain sends a protected message without a keyboard.
func (c *StepContext) SendPlain(text string) (*tgbotapi.Message, error) {
	return c.Send(text, nil)
}

// SendReply sends a protected reply to a specific message.
func (c *StepContext) SendReply(replyTo int, text string) (*tgbotapi.Message, error) {
	newMsg, err := SendProtectedMessage(c.Opts.bot, c.ChatID, replyTo, false, text, c.Ecode)
	if err != nil {
		if IsForbiddenError(err) {
			c.Ban()
			return nil, fmt.Errorf("forbidden: %w", err)
		}
		return nil, err
	}
	return newMsg, nil
}

// Transition moves the user to a new step, sends that step's message, and saves the session.
// payload is optional data to store (e.g., the brigadier name for restore_words).
func (c *StepContext) Transition(targetStepID StepID, state int, payload []byte) error {
	step := GetStep(targetStepID)
	if step == nil {
		return fmt.Errorf("unknown step: %q", targetStepID)
	}

	// If the step has a custom OnEnter, use that.
	if step.OnEnter != nil {
		// Pre-set the stage in session so OnEnter can reference it.
		c.Session.Stage = StepIDToLegacyStage[targetStepID]
		c.Session.State = state
		c.Session.Payload = payload
		return step.OnEnter(c)
	}

	// Default: send the flow message + keyboard, then save session.
	text := c.FlowMessage(step.FlowStage, step.FallbackMessage)
	kb := c.FlowKeyboard(step.FlowStage, step.FallbackKeyboard)

	newMsg, err := c.Send(text, &kb)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "[d:%s] transitioned to step %q with state %d\n", c.Ecode, targetStepID, state)
	return c.SaveSession(newMsg, StepIDToLegacyStage[targetStepID], state, payload)
}

// SaveSession persists session state after sending a message.
func (c *StepContext) SaveSession(sentMsg *tgbotapi.Message, stage, state int, payload []byte) error {
	return setSession(
		c.Opts.db, c.Opts.sessionSecret,
		c.Session.Label, &c.Session.Captcha,
		sentMsg.Chat.ID, sentMsg.MessageID, int64(sentMsg.Date),
		stage, state, payload,
	)
}

// SaveSessionNoMsg persists session state without a new message.
func (c *StepContext) SaveSessionNoMsg(stage, state int, payload []byte) error {
	return setSession(
		c.Opts.db, c.Opts.sessionSecret,
		c.Session.Label, &c.Session.Captcha,
		c.ChatID, 0, 0,
		stage, state, payload,
	)
}

// Ban sets the session to the banned state.
func (c *StepContext) Ban() {
	setSession(c.Opts.db, c.Opts.sessionSecret,
		c.Session.Label, &c.Session.Captcha,
		c.ChatID, 0, 0,
		stageMainTrackCleanup, SessionStateBanOnBan, nil)
}

// Wrong shows a generic error message to the user.
func (c *StepContext) Wrong(err error) {
	stWrong(c.Opts.bot, c.ChatID, c.Ecode, err, c.Lang)
}

// RemovePreviousKeyboard removes the inline keyboard from the previous message.
func (c *StepContext) RemovePreviousKeyboard(text string) {
	if c.Session.OurMsgID == 0 {
		return
	}
	if err := RemoveKeyboardMsg(c.Opts.bot, c.ChatID, c.Session.OurMsgID, text); err != nil {
		logs.Errf("[!:%s] remove keyboard: %s\n", c.Ecode, err)
	}
}

// RemoveMessage deletes a message.
func (c *StepContext) RemoveMessage(msgID int) {
	if err := RemoveMsg(c.Opts.bot, c.ChatID, msgID); err != nil {
		logs.Errf("[!:%s] remove: %s\n", c.Ecode, err)
	}
}

// CheckMaintenance checks maintenance mode and sends a message if active.
// Returns true if maintenance is active (caller should return).
func (c *StepContext) CheckMaintenance(whenfull bool) bool {
	return checkMaintenanceMode(c.Opts, c.Session.Label, &c.Session.Captcha, c.ChatID, c.Ecode, whenfull)
}

// --- Dispatcher ---

// DispatchMessage is the new entrypoint for handling messages.
func DispatchMessage(opts handlerOpts, update tgbotapi.Update, dept MinistryOpts) {
	defer opts.wg.Done()

	ecode := genEcode()
	lang := userLang(update.Message.From.LanguageCode)

	if update.Message.ForwardFrom != nil || update.Message.ForwardFromChat != nil {
		SendProtectedMessage(opts.bot, update.Message.Chat.ID, 0, false,
			flowMessage(opts.flowMainUrl, "forbid_forwards", InfoForbidForwardsMessage, lang), ecode)
		return
	}

	session, ok := auth(opts, update.Message.Chat.ID, update.Message.Date, ecode, lang)
	if !ok {
		return
	}
	defer opts.cw.Release(update.Message.Chat.ID)

	fmt.Fprintf(os.Stderr, "[d:%s] received message in stage %d\n", ecode, session.Stage)

	time.Sleep(SlowAnswerTimeout)

	ctx := &StepContext{
		Opts:    opts,
		Session: session,
		ChatID:  update.Message.Chat.ID,
		Ecode:   ecode,
		Lang:    lang,
		Dept:    dept,
	}

	// Handle commands first.
	if update.Message.IsCommand() {
		if err := dispatchCommand(ctx, update.Message); err != nil {
			if IsForbiddenError(err) {
				ctx.Ban()
				return
			}
			ctx.Wrong(fmt.Errorf("command: %s: %w", update.Message.Command(), err))
		}
		return
	}

	// Resolve current step.
	stepID, ok := LegacyStageToStepID[session.Stage]
	if !ok {
		ctx.Wrong(fmt.Errorf("unknown stage: %d", session.Stage))
		return
	}

	step := GetStep(stepID)
	if step == nil {
		ctx.Wrong(fmt.Errorf("unregistered step: %q", stepID))
		return
	}

	if step.OnMessage == nil {
		// Step doesn't expect messages — show a fallback.
		SendProtectedMessage(opts.bot, ctx.ChatID, update.Message.MessageID, false,
			ctx.FlowMessage("unknown_command", InfoUnknownCommandMessage), ecode)
		return
	}

	if err := step.OnMessage(ctx, update.Message); err != nil {
		if IsForbiddenError(err) {
			ctx.Ban()
			return
		}
		ctx.Wrong(fmt.Errorf("%s: %w", stepID, err))
	}
}

// DispatchCallback is the new entrypoint for handling button callbacks.
func DispatchCallback(opts handlerOpts, update tgbotapi.Update, dept MinistryOpts) {
	defer opts.wg.Done()

	ecode := genEcode()
	lang := userLang(update.CallbackQuery.From.LanguageCode)

	session, ok := auth(opts, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.Date, ecode, lang)
	if !ok {
		return
	}
	defer opts.cw.Release(update.CallbackQuery.Message.Chat.ID)

	time.Sleep(SlowAnswerTimeout)

	ctx := &StepContext{
		Opts:    opts,
		Session: session,
		ChatID:  update.CallbackQuery.Message.Chat.ID,
		Ecode:   ecode,
		Lang:    lang,
		Dept:    dept,
	}

	cbData := update.CallbackQuery.Data
	cbMsgID := update.CallbackQuery.Message.MessageID

	fmt.Fprintf(os.Stderr, "[d:%s] received callback in stage %d with data %q\n", ecode, session.Stage, cbData)

	// Handle global callbacks that can be triggered from any state.
	ctx.Session.OurMsgID = cbMsgID
	switch cbData {
	case "outline_download_urls":
		if err := sendDownloadOutlineMessage(ctx.Opts.bot, ctx.ChatID, ctx.Lang, ctx.Opts.flowMainUrl); err != nil {
			if IsForbiddenError(err) {
				ctx.Ban()
				return
			}
			ctx.Wrong(fmt.Errorf("outline: %w", err))
		}
		return
	case "amnezia_vpn_download_urls":
		if err := sendDownloadAmneziaVPNMessage(ctx.Opts.bot, ctx.ChatID, ctx.Opts.flowMainUrl); err != nil {
			if IsForbiddenError(err) {
				ctx.Ban()
				return
			}
			ctx.Wrong(fmt.Errorf("amnezia: %w", err))
		}
		return
	case "reset":
		if err := stepResetCallback(ctx); err != nil {
			if IsForbiddenError(err) {
				ctx.Ban()
				return
			}
			ctx.Wrong(fmt.Errorf("reset: %w", err))
		}
		return
	}

	// Resolve current step.
	stepID, ok := LegacyStageToStepID[session.Stage]
	if !ok {
		logs.Debugf("[!:%s] unknown stage %d for callback %q\n", ecode, session.Stage, cbData)
		return
	}

	step := GetStep(stepID)
	if step == nil {
		logs.Debugf("[!:%s] unregistered step %q for callback %q\n", ecode, stepID, cbData)
		return
	}

	if step.OnCallback == nil {
		logs.Errf("no callback handler for step %q data %q\n", stepID, cbData)
		return
	}

	// Store callback message ID so handlers can reference it.
	ctx.Session.OurMsgID = cbMsgID

	logs.Debugf("[d:%s] dispatching callback for step %q with data %q\n", ecode, stepID, cbData)
	if err := step.OnCallback(ctx, cbData); err != nil {
		if IsForbiddenError(err) {
			ctx.Ban()
			return
		}
		ctx.Wrong(fmt.Errorf("%s callback %q: %w", stepID, cbData, err))
	}
}

// DispatchReaction is the new entrypoint for handling emoji reactions.
func DispatchReaction(opts handlerOpts, update tgbotapi.Update) {
	defer opts.wg.Done()

	ecode := genEcode()

	lang := langRU
	if update.MessageReaction.User != nil {
		lang = userLang(update.MessageReaction.User.LanguageCode)
	}

	if update.MessageReaction.Chat.Type != "private" {
		SendProtectedMessage(opts.bot, update.MessageReaction.Chat.ID, 0, false,
			flowMessage(opts.flowMainUrl, "forbid_forwards", InfoForbidForwardsMessage, lang), ecode)
		return
	}

	session, ok := auth(opts, update.MessageReaction.Chat.ID, update.MessageReaction.Date, ecode, lang)
	if !ok {
		return
	}
	defer opts.cw.Release(update.MessageReaction.Chat.ID)

	time.Sleep(SlowAnswerTimeout)

	ctx := &StepContext{
		Opts:    opts,
		Session: session,
		ChatID:  update.MessageReaction.Chat.ID,
		Ecode:   ecode,
		Lang:    lang,
	}

	// Captcha reaction check (cross-step, always active if captcha not passed).
	c := session.Captcha
	if !c.Passed && c.MessageID == update.MessageReaction.MessageID {
		for _, r := range update.MessageReaction.NewReaction {
			if r.Type == "emoji" && r.Emoji == c.Reaction {
				c.Passed = true
				if err := sendSuccessLikeV2(ctx, &c); err != nil {
					ctx.Wrong(fmt.Errorf("success like: %w", err))
				}
				return
			}
		}
	}
}

func sendSuccessLikeV2(ctx *StepContext, c *SessionCaptcha) error {
	text := ctx.FlowMessage("captcha_passed", "Проверка пройдена! Нажми /repeat для продолжения.")
	newMsg, err := ctx.SendPlain(text)
	if err != nil {
		return err
	}
	return setSession(ctx.Opts.db, ctx.Opts.sessionSecret, ctx.Session.Label, c,
		newMsg.Chat.ID, newMsg.MessageID, int64(newMsg.Date),
		ctx.Session.Stage, ctx.Session.State, nil)
}

// --- Command dispatch ---

func dispatchCommand(ctx *StepContext, msg *tgbotapi.Message) error {
	command := msg.Command()
	logs.Debugf("[d:%s] command: %s, stage: %d\n", ctx.Ecode, command, ctx.Session.Stage)

	// Debug reset.
	if ctx.Opts.debug == int(logs.LevelDebug) && command == "vpnregen" {
		if err := resetSession(ctx.Opts.db, ctx.Opts.sessionSecret, ctx.ChatID); err != nil {
			return fmt.Errorf("vpnregen: %w", err)
		}
		ctx.SendReply(0, ctx.FlowMessage("reset_success", MainTrackResetSuccessfull))
		return nil
	}

	switch command {
	case "start":
		return cmdStart(ctx, msg)
	case "restore":
		return cmdRestore(ctx, msg)
	case "repeat":
		return cmdRepeat(ctx, msg)
	default:
		return cmdDefault(ctx, msg)
	}
}

func cmdStart(ctx *StepContext, msg *tgbotapi.Message) error {
	s := msg.CommandArguments()
	logs.Debugf("[d:%s] start args: %q, %d\n", ctx.Ecode, s, len(s))

	// Check if it's a UUID for custom VIP brigade.
	if len(s) == 36 {
		if _, err := uuid.Parse(s); err == nil {
			requestID, err := reqBrigade(ctx.Dept, ctx.ChatID, ctx.Session.Label, s)
			if err != nil || requestID == uuid.Nil {
				ctx.Wrong(fmt.Errorf("request custom brigade failed"))
				return nil
			}

			text := ctx.FlowMessage("vip_welcome", MainTrackVIPWelcomeMessage)
			newMsg, err := ctx.SendPlain(text)
			if err != nil {
				return fmt.Errorf("custom vip: %w", err)
			}
			return ctx.SaveSession(newMsg, stageMainTrackStart, SessionStatePayloadSomething, nil)
		}
	}

	return cmdStartWelcome(ctx, msg)
}

func cmdRestore(ctx *StepContext, msg *tgbotapi.Message) error {
	if ctx.CheckMaintenance(true) {
		return nil
	}

	prev := 0
	if ctx.Session.Stage == stageMainTrackCleanup || ctx.Session.Stage == stageRestoreTrackStart ||
		ctx.Session.Stage == stageRestoreTrackSendName || ctx.Session.Stage == stageRestoreTrackSendWords ||
		ctx.Session.Stage == stageRestoreTrackCleanup {
		prev = ctx.Session.State
	}

	if prev == SessionStatePayloadBan {
		ctx.SendReply(msg.MessageID, ctx.FlowMessage("conversation_finished", MainTrackWarnConversationsFinished))
		return nil
	}

	time.Sleep(SlowAnswerTimeout)

	return ctx.Transition("restore_start", prev, nil)
}

func cmdRepeat(ctx *StepContext, msg *tgbotapi.Message) error {
	if ctx.CheckMaintenance(true) {
		return nil
	}

	switch ctx.Session.Stage {
	case stageMainTrackWaitForBill:
		return stepQuizEnter(ctx)
	case stageMainTrackCleanup:
		ctx.SendReply(msg.MessageID, ctx.FlowMessage("repeat_conversation_finished", RepeatTrackWarnConversationsFinished))
		return nil
	}

	// For all other stages, fall through to default behavior.
	return cmdDefault(ctx, msg)
}

func cmdDefault(ctx *StepContext, msg *tgbotapi.Message) error {
	stepID, ok := LegacyStageToStepID[ctx.Session.Stage]
	if !ok {
		return nil
	}

	switch stepID {
	case "restore_words":
		fmt.Fprintf(os.Stderr, "[d:%s] /repeat in restore_words with prev state %d\n", ctx.Ecode, ctx.Session.State)
		if ctx.CheckMaintenance(true) {
			return nil
		}
		return ctx.Transition("restore_words", ctx.Session.State, ctx.Session.Payload)
	case "restore_name":
		fmt.Fprintf(os.Stderr, "[d:%s] /repeat in restore_name with prev state %d\n", ctx.Ecode, ctx.Session.State)
		if ctx.CheckMaintenance(true) {
			return nil
		}
		return ctx.Transition("restore_name", ctx.Session.State, nil)
	case "restore_start":
		fmt.Fprintf(os.Stderr, "[d:%s] /repeat in restore_start with prev state %d\n", ctx.Ecode, ctx.Session.State)
		if ctx.CheckMaintenance(true) {
			return nil
		}
		return ctx.Transition("restore_start", ctx.Session.State, nil)
	case "cleanup":
		ctx.SendReply(msg.MessageID, ctx.FlowMessage("conversation_finished", MainTrackWarnConversationsFinished))
		return nil
	case "wait_for_approval":
		if ctx.CheckMaintenance(false) {
			return nil
		}
		ctx.SendReply(msg.MessageID, ctx.FlowMessage("warn_wait_approval", MainTrackWarnWaitForApprovement))
		return nil
	case "main_start", "welcome":
		fmt.Fprintf(os.Stderr, "[d:%s] /repeat in %q with prev state %d\n", ctx.Ecode, stepID, ctx.Session.State)
		return cmdStartWelcome(ctx, msg)
	case "wait_for_bill":
		if ctx.CheckMaintenance(false) {
			return nil
		}
		return stepBillOnMessage(ctx, msg)
	default:
		fmt.Fprintf(os.Stderr, "[d:%s] unrecognized /repeat stage %d\n", ctx.Ecode, ctx.Session.Stage)
		if ctx.CheckMaintenance(false) {
			return nil
		}
		ctx.SendReply(0, ctx.FlowMessage("unknown_command", InfoUnknownCommandMessage))
		return nil
	}
}

func cmdStartWelcome(ctx *StepContext, msg *tgbotapi.Message) error {
	label := onlyBase64Symbols.ReplaceAllString(msg.CommandArguments(), "")
	if len(label) > 64 {
		label = label[:64]
	}

	sessionLabel := SessionLabel{
		Label: label,
		Time:  time.Now(),
		ID:    uuid.New(),
	}

	if ctx.Session.Label.Label != "" && !ctx.Session.Label.Time.IsZero() && ctx.Session.Label.ID != uuid.Nil {
		sessionLabel = ctx.Session.Label
	}

	command := msg.Command()

	fmt.Fprintf(os.Stderr, "[d:%s] start with label %q (session has label %q), command: %s\n", ctx.Ecode, sessionLabel.Label, ctx.Session.Label.Label, command)
	if command == "start" && ctx.Session.Stage == stageMainTrackStart {
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

		if err := ctx.Opts.ls.Update(sessionLabel); err != nil {
			return fmt.Errorf("update label: %w", err)
		}
	}

	if ctx.Session.Stage != stageMainTrackWaitForWanting && !warnAutodeleteSettings(ctx.Opts, ctx.ChatID, ctx.Ecode) {
		ctx.Session.Label = setLabel(ctx.Session.Label, MarkerEmptyLabel)
		setSession(ctx.Opts.db, ctx.Opts.sessionSecret, sessionLabel, &ctx.Session.Captcha,
			ctx.ChatID, 0, 0, stageMainTrackStart, SessionStatePayloadBan, nil)
		return nil
	}

	if !checkCaptcha(ctx.Opts, &ctx.Session.Captcha, ctx.Session.Label, ctx.ChatID, ctx.Ecode, ctx.Session.Stage, ctx.Session.State, ctx.Session.Payload) {
		return nil
	}

	// Temporarily override session label for the welcome message.
	origLabel := ctx.Session.Label
	ctx.Session.Label = sessionLabel
	err := ctx.Transition("welcome", SessionStatePayloadSomething, nil)
	if err != nil {
		ctx.Session.Label = origLabel
	}
	return err
}
