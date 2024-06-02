package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	getOutlineForAndroid = "Скачать для Android"
	getOutlineForIOS     = "Скачать для iOS"
	getOutlineForChrome  = "Скачать для Chrome"
	getOutlineForWindows = "Скачать для Windows"
	getOutlineForMacOS   = "Скачать для macOS"
	getOutlineForLinux   = "Скачать для Linux"
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
	var buttons []tgbotapi.InlineKeyboardButton

	for _, title := range outlineDownloadArray {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonURL(title, outlineDownloadURLMap[title]))
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons)
}()

const outlineDownloadMessage = `Для использования Outline скачай и установи приложение для своей платформы:`

// SendDownloadOutlineMessage - send message with download links for Outline.
func sendDownloadOutlineMessage(opts handlerOpts, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, outlineDownloadMessage)
	msg.ReplyMarkup = outlineDownloadKeyboard
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = false

	if _, err := opts.bot.Send(msg); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return nil
}
