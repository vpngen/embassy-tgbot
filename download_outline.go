package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	getOutlineForAndroid = "для Android"
	getOutlineForIOS     = "для iOS"
	getOutlineForChrome  = "для Chrome"
	getOutlineForWindows = "для Windows"
	getOutlineForMacOS   = "для macOS"
	getOutlineForLinux   = "для Linux"
)

var outlineDownloadURLMap = map[string]string{
	getOutlineForAndroid: "https://play.google.com/store/apps/details?id=org.outline.android.client",
	getOutlineForIOS:     "https://itunes.apple.com/us/app/outline-app/id1356177741",
	getOutlineForChrome:  "https://play.google.com/store/apps/details?id=org.outline.android.client",
	getOutlineForWindows: "https://s3.amazonaws.com/outline-releases/client/windows/stable/Outline-Client.exe",
	getOutlineForMacOS:   "https://itunes.apple.com/us/app/outline-app/id1356178125",
	getOutlineForLinux:   "https://s3.amazonaws.com/outline-releases/client/linux/stable/Outline-Client.AppImage",
}

var outlineDownloadArray = []string{
	getOutlineForAndroid,
	getOutlineForIOS,
	getOutlineForWindows,
	getOutlineForMacOS,
	getOutlineForLinux,
	getOutlineForChrome,
}

var outlineDownloadKeyboard = func() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, title := range outlineDownloadArray {
		rows = append(rows,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(title, outlineDownloadURLMap[title]),
			),
		)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}()

var outlineDownloadKeyboardShort = func() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(getOutlineForAndroid, outlineDownloadURLMap[getOutlineForAndroid]),
			tgbotapi.NewInlineKeyboardButtonURL(getOutlineForIOS, outlineDownloadURLMap[getOutlineForIOS]),
			tgbotapi.NewInlineKeyboardButtonURL(getOutlineForWindows, outlineDownloadURLMap[getOutlineForWindows]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("для других платформ", "outline_download_urls"),
		),
	)
}()

const outlineDownloadMessage = `Для использования Outline скачай и установи приложение для своей платформы:`

// SendDownloadOutlineMessage - send message with download links for Outline.
func sendDownloadOutlineMessage(bot *tgbotapi.BotAPI, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, outlineDownloadMessage)
	msg.ReplyMarkup = outlineDownloadKeyboard
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = false

	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return nil
}

var outlineDownloadMessageShort = `Осталось три простых шага до свободного интернета! 
*Шаг 1.* Скачай и установи Outline.`

// SendDownloadOutlineMessageShort - send message with download links for Outline.
func sendDownloadOutlineMessageShort(bot *tgbotapi.BotAPI, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, outlineDownloadMessageShort)
	msg.ReplyMarkup = outlineDownloadKeyboardShort
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = false

	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return nil
}
