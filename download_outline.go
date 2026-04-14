package main

import (
	"fmt"

	tgbotapi "github.com/vpngen/embassy-tgbot/telegram-bot-api"
)

const (
	getOutlineForAndroid = "для Android"
	getOutlineForIOS     = "для iOS"
	getOutlineForChrome  = "для Chrome"
	getOutlineForWindows = "для Windows"
	getOutlineForMacOS   = "для macOS"
	getOutlineForLinux   = "для Linux"
)

const (
	getOutlineForAndroidEN = "for Android"
	getOutlineForIOSEN     = "for iOS"
	getOutlineForChromeEN  = "for Chrome"
	getOutlineForWindowsEN = "for Windows"
	getOutlineForMacOSEN   = "for macOS"
	getOutlineForLinuxEN   = "for Linux"
)

var outlineDownloadURLMap = map[string]string{
	getOutlineForAndroid: "https://play.google.com/store/apps/details?id=org.outline.android.client",
	getOutlineForIOS:     "https://itunes.apple.com/us/app/outline-app/id1356177741",
	getOutlineForChrome:  "https://play.google.com/store/apps/details?id=org.outline.android.client",
	getOutlineForWindows: "https://s3.amazonaws.com/outline-releases/client/windows/stable/Outline-Client.exe",
	getOutlineForMacOS:   "https://itunes.apple.com/us/app/outline-app/id1356178125",
	getOutlineForLinux:   "https://s3.amazonaws.com/outline-releases/client/linux/stable/Outline-Client.AppImage",
}

var outlineDownloadURLMapEN = map[string]string{
	getOutlineForAndroidEN: "https://play.google.com/store/apps/details?id=org.outline.android.client",
	getOutlineForIOSEN:     "https://itunes.apple.com/us/app/outline-app/id1356177741",
	getOutlineForChromeEN:  "https://play.google.com/store/apps/details?id=org.outline.android.client",
	getOutlineForWindowsEN: "https://s3.amazonaws.com/outline-releases/client/windows/stable/Outline-Client.exe",
	getOutlineForMacOSEN:   "https://itunes.apple.com/us/app/outline-app/id1356178125",
	getOutlineForLinuxEN:   "https://s3.amazonaws.com/outline-releases/client/linux/stable/Outline-Client.AppImage",
}

var outlineDownloadArray = []string{
	getOutlineForAndroid,
	getOutlineForIOS,
	getOutlineForWindows,
	getOutlineForMacOS,
	getOutlineForLinux,
	getOutlineForChrome,
}

var outlineDownloadArrayEN = []string{
	getOutlineForAndroidEN,
	getOutlineForIOSEN,
	getOutlineForWindowsEN,
	getOutlineForMacOSEN,
	getOutlineForLinuxEN,
	getOutlineForChromeEN,
}

func buildOutlineDownloadKeyboard(lang, flowMainUrl string) tgbotapi.InlineKeyboardMarkup {
	arr := outlineDownloadArray
	urlMap := outlineDownloadURLMap
	if lang == langEN {
		arr = outlineDownloadArrayEN
		urlMap = outlineDownloadURLMapEN
	}

	urlMap = ministryDownloadURLs(flowMainUrl, "outline", lang, urlMap)

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, title := range arr {
		rows = append(rows,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(title, urlMap[title]),
			),
		)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func buildOutlineDownloadKeyboardShort(lang, flowMainUrl string) tgbotapi.InlineKeyboardMarkup {
	if lang == langEN {
		urlMap := ministryDownloadURLs(flowMainUrl, "outline", lang, outlineDownloadURLMapEN)
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(getOutlineForAndroidEN, urlMap[getOutlineForAndroidEN]),
				tgbotapi.NewInlineKeyboardButtonURL(getOutlineForIOSEN, urlMap[getOutlineForIOSEN]),
				tgbotapi.NewInlineKeyboardButtonURL(getOutlineForWindowsEN, urlMap[getOutlineForWindowsEN]),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("for other platforms", "outline_download_urls"),
			),
		)
	}

	urlMap := ministryDownloadURLs(flowMainUrl, "outline", lang, outlineDownloadURLMap)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(getOutlineForAndroid, urlMap[getOutlineForAndroid]),
			tgbotapi.NewInlineKeyboardButtonURL(getOutlineForIOS, urlMap[getOutlineForIOS]),
			tgbotapi.NewInlineKeyboardButtonURL(getOutlineForWindows, urlMap[getOutlineForWindows]),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("для других платформ", "outline_download_urls"),
		),
	)
}

const outlineDownloadMessageRU = `Для использования Outline скачай и установи приложение для своей платформы
` + "\u2139\ufe0f" + ` [@vpngen](http://t.me/vpngen)`

const outlineDownloadMessageEN = `To use Outline, download and install the app for your platform
` + "\u2139\ufe0f" + ` [@vpngen](http://t.me/vpngen)`

// sendDownloadOutlineMessage - send message with download links for Outline.
func sendDownloadOutlineMessage(bot *tgbotapi.BotAPI, chatID int64, lang, flowMainUrl string) error {
	text := ministryMessage(flowMainUrl, "outline_download_full", outlineDownloadMessageRU, lang)
	if lang == langEN && text == outlineDownloadMessageRU {
		text = outlineDownloadMessageEN
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = buildOutlineDownloadKeyboard(lang, flowMainUrl)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = false

	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return nil
}

var outlineDownloadMessageShortRU = `Осталось три простых шага до свободного интернета! 
*Шаг 1.* Скачай и установи Outline.`

var outlineDownloadMessageShortEN = `Three simple steps to a free internet! 
*Step 1.* Download and install Outline.`

// sendDownloadOutlineMessageShort - send message with download links for Outline.
func sendDownloadOutlineMessageShort(bot *tgbotapi.BotAPI, chatID int64, lang, flowMainUrl string) error {
	text := ministryMessage(flowMainUrl, "outline_download_short", outlineDownloadMessageShortRU, lang)
	if lang == langEN && text == outlineDownloadMessageShortRU {
		text = outlineDownloadMessageShortEN
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = buildOutlineDownloadKeyboardShort(lang, flowMainUrl)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = false

	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return nil
}
