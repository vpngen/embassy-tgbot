package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/vpngen/embassy-tgbot/telegram-bot-api"
)

const (
	langRU = "ru"
	langEN = "en"
)

// httpClient is a shared client with a reasonable timeout for admin-panel requests.
var httpClient = &http.Client{Timeout: 5 * time.Second}

// adminAPIKey is set at startup from config; sent as X-API-Key header to admin-panel.
var adminAPIKey string

// userLang returns "en" only for English users, "ru" for everyone else.
func userLang(tgLangCode string) string {
	if strings.HasPrefix(strings.ToLower(tgLangCode), "en") {
		fmt.Fprintf(os.Stderr, "[debug] userLang: tgLangCode=%q -> en\n", tgLangCode)
		return langEN
	}
	fmt.Fprintf(os.Stderr, "[debug] userLang: tgLangCode=%q -> ru\n", tgLangCode)
	return langRU
}

// langQuery appends ?lang=en for English, returns the URL unchanged for Russian.
func langQuery(baseURL, lang string) string {
	if lang == "" || lang == langRU {
		return baseURL
	}
	if strings.Contains(baseURL, "?") {
		return baseURL + "&lang=" + lang
	}
	return baseURL + "?lang=" + lang
}

type flowRuntime struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Stages []flowRuntimeStep `json:"stages"`
}

type flowRuntimeStep struct {
	ID      string              `json:"id"`
	Message string              `json:"message"`
	Buttons []flowRuntimeButton `json:"buttons,omitempty"`
}

type flowRuntimeButton struct {
	Label  string `json:"label"`
	Action string `json:"action"`
	Target string `json:"target"`
}

func fetchJSON(url string, target any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request %s: %w", url, err)
	}
	if adminAPIKey != "" {
		req.Header.Set("X-API-Key", adminAPIKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http get %s: status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body %s: %w", url, err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("parse json %s: %w", url, err)
	}

	return nil
}

func fetchFlowRuntime(baseURL, lang string) (*flowRuntime, error) {
	url := langQuery(baseURL, lang)
	var flow flowRuntime
	if err := fetchJSON(url, &flow); err != nil {
		return nil, err
	}
	return &flow, nil
}

func flowMessage(flowURL, stageID, fallback, lang string) string {
	fmt.Fprintf(os.Stderr, "[debug] flowMessage: lang=%q url=%q stageID=%q\n", lang, flowURL, stageID)
	flow, err := fetchFlowRuntime(flowURL, lang)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[debug] flowMessage: fetch err=%v, using fallback\n", err)
		return fallback
	}

	for _, s := range flow.Stages {
		if s.ID == stageID && s.Message != "" {
			return s.Message
		}
	}

	return fallback
}

// ministryMessagesFile holds the parsed contents of a ministry JSON response.
type ministryMessagesFile struct {
	Messages  map[string]string            `json:"messages"`
	Downloads map[string]map[string]string `json:"downloads,omitempty"`
}

// deriveMinistryURL converts a flow URL like "http://host:port/api/flows/main"
// to the ministry endpoint "http://host:port/api/ministry".
func deriveMinistryURL(flowURL string) string {
	if idx := strings.Index(flowURL, "/api/flows/"); idx != -1 {
		return flowURL[:idx] + "/api/ministry"
	}
	// fallback: try trimming last path segment and appending ministry
	if idx := strings.LastIndex(flowURL, "/"); idx != -1 {
		return flowURL[:idx] + "/ministry"
	}
	return flowURL
}

// fetchMinistryMessages fetches ministry messages from the admin-panel API.
func fetchMinistryMessages(flowURL, lang string) (*ministryMessagesFile, error) {
	url := langQuery(deriveMinistryURL(flowURL), lang)
	var mf ministryMessagesFile
	if err := fetchJSON(url, &mf); err != nil {
		return nil, err
	}
	return &mf, nil
}

// ministryDownloadURLs fetches download URLs from the admin-panel ministry API.
// Returns the fallback map if the API is unreachable or the key is missing.
func ministryDownloadURLs(flowURL, key, lang string, fallback map[string]string) map[string]string {
	mf, err := fetchMinistryMessages(flowURL, lang)
	if err != nil {
		return fallback
	}

	if urls, ok := mf.Downloads[key]; ok && len(urls) > 0 {
		return urls
	}

	return fallback
}

// ministryMessage fetches a message from the admin-panel ministry API by key.
// Falls back to the hardcoded constant if the API or key is unavailable.
func ministryMessage(flowURL, key, fallback, lang string) string {
	mf, err := fetchMinistryMessages(flowURL, lang)
	if err != nil {
		return fallback
	}

	if msg, ok := mf.Messages[key]; ok && msg != "" {
		return msg
	}

	return fallback
}

func flowKeyboard(flowURL, stageID, supportURL, lang string) (*tgbotapi.InlineKeyboardMarkup, bool) {
	flow, err := fetchFlowRuntime(flowURL, lang)
	if err != nil {
		return nil, false
	}

	var stage *flowRuntimeStep
	for i := range flow.Stages {
		if flow.Stages[i].ID == stageID {
			stage = &flow.Stages[i]
			break
		}
	}
	if stage == nil || len(stage.Buttons) == 0 {
		return nil, false
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(stage.Buttons))
	for _, b := range stage.Buttons {
		switch b.Action {
		case "url":
			url := strings.ReplaceAll(b.Target, "{{support_url}}", supportURL)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL(b.Label, url)))
		case "goto", "call":
			cb, ok := flowTargetToCallback(b.Action, b.Target)
			if !ok {
				continue
			}
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(b.Label, cb)))
		}
	}

	if len(rows) == 0 {
		return nil, false
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)

	return &kb, true
}

func flowTargetToCallback(action, target string) (string, bool) {
	switch action {
	case "goto":
		switch target {
		case "captcha":
			return "started", true
		case "restore_start", "restore_name", "restore_words":
			return "restore", true
		case "welcome":
			return "reset", true
		default:
			return "", false
		}
	case "call":
		switch target {
		case "vip_redirect":
			return "vip", true
		default:
			return target, target != ""
		}
	default:
		return "", false
	}
}

// decisionsFile mirrors the admin-panel decisions.json structure.
type decisionsFile struct {
	Decisions []struct {
		Code           int    `json:"code"`
		Key            string `json:"key"`
		Button         string `json:"button"`
		Template       string `json:"template"`
		HasSupportLink bool   `json:"has_support_link"`
	} `json:"decisions"`
	SupportLinkTemplate string `json:"support_link_template"`
}

// flowDecisionComments fetches decisions from the admin-panel API and renders templates
// with the given support URL. Returns a map[int]string keyed by decision code.
// On any error it returns nil so the caller can fall back to hardcoded values.
func flowDecisionComments(decisionsURL, supportURL, lang string) map[int]string {
	url := langQuery(decisionsURL, lang)
	var df decisionsFile
	if err := fetchJSON(url, &df); err != nil {
		return nil
	}

	// Render the support link sentence itself.
	supportLink := strings.ReplaceAll(df.SupportLinkTemplate, "{{support_url}}", supportURL)

	out := make(map[int]string, len(df.Decisions))
	for _, d := range df.Decisions {
		t := d.Template
		if d.HasSupportLink {
			t = strings.ReplaceAll(t, "{{support_link}}", supportLink)
		}
		t = strings.ReplaceAll(t, "{{support_url}}", supportURL)
		out[d.Code] = t
	}

	return out
}
