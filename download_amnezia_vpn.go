package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	getAmneziaVPNForAndroid = "для Android"
	getAmneziaVPNForIOS     = "для iOS"
	getAmneziaVPNForWindows = "для Windows"
	getAmneziaVPNForMacOS   = "для macOS"
	getAmneziaVPNForLinux   = "для Linux"
)

var amneziaVPNDownloadURLMap = map[string]string{
	getAmneziaVPNForAndroid: "https://play.google.com/store/apps/details?id=org.amnezia.vpn",
	getAmneziaVPNForIOS:     "https://apps.apple.com/us/app/amneziavpn/id1600529900",
	getAmneziaVPNForWindows: "https://github.com/amnezia-vpn/amnezia-client/releases/download/4.5.3.0/AmneziaVPN_4.5.3.0_x64.exe",
	getAmneziaVPNForMacOS:   "https://github.com/amnezia-vpn/amnezia-client/releases/download/4.5.3.0/AmneziaVPN_4.5.3.0.dmg",
	getAmneziaVPNForLinux:   "https://github.com/amnezia-vpn/amnezia-client/releases/download/4.5.3.0/AmneziaVPN_Linux_installer.tar.zip",
}

var amneziaVPNDownloadArray = []string{
	getAmneziaVPNForAndroid,
	getAmneziaVPNForIOS,
	getAmneziaVPNForWindows,
	getAmneziaVPNForMacOS,
	getAmneziaVPNForLinux,
}

var amneziaVPNDownloadKeyboard = func() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, title := range amneziaVPNDownloadArray {
		rows = append(rows,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL(title, amneziaVPNDownloadURLMap[title]),
			),
		)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}()

var amneziaVPNDownloadKeyboardShort = func() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("скачать AmneziaVPN", "amnezia_vpn_download_urls"),
		),
	)
}()

const amneziaVPNDownloadMessage = `Для использования AmneziaVPN скачай и установи приложение для своей платформы:`

// SendDownloadAmmneziaVPNMessage - send message with download links for Outline.
func sendDownloadAmneziaVPNMessage(bot *tgbotapi.BotAPI, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, amneziaVPNDownloadMessage)
	msg.ReplyMarkup = amneziaVPNDownloadKeyboard
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true
	msg.ProtectContent = false

	if _, err := bot.Send(msg); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return nil
}
