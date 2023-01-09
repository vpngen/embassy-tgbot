package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	acceptReceiptPrefix = "a"
	rejectReceiptPrefix = "r"
)

// receipt check decisions
const (
	decisionUnknown int = iota
	decisionAcceptGeneral
	decisionAcceptCats
	decisionRejectUnacceptable
	decisionRejectUnreadable
	decisionRejectElectronic
	decisionRejectIncomplete
	decisionRejectUnverifiable
	decisionRejectAmountMismatch
	decisionRejectTooOld
	decisionRejectWithCallback
)

var (
	buttons = map[int]string{
		decisionUnknown:              "\U0001f937 Не знаю",
		decisionAcceptGeneral:        "\U0001f44d Подтвердить",
		decisionAcceptCats:           "\U0001f63b Котики",
		decisionRejectUnacceptable:   "\U0001f595 Бан", //\U0000274c
		decisionRejectUnreadable:     "\U0001f576 Нечит.",
		decisionRejectElectronic:     "\U0001f4f1 Эл-ый",
		decisionRejectIncomplete:     "\U0001f313 Неполн.",
		decisionRejectUnverifiable:   "\U0001f46e Непров.",
		decisionRejectAmountMismatch: "\U0001f4b5 Сумма",
		decisionRejectTooOld:         "\U0001f4c5 Старый",
		decisionRejectWithCallback:   "\u260e\ufe0f Свяжитесь",
	}
)

func makeKeyboardButton(reason int, id string, accept bool) tgbotapi.InlineKeyboardButton {
	var payload string

	switch accept {
	case true:
		payload = fmt.Sprintf("%s%d-%s", acceptReceiptPrefix, reason, id)
	default:
		payload = fmt.Sprintf("%s%d-%s", rejectReceiptPrefix, reason, id)
	}

	return tgbotapi.NewInlineKeyboardButtonData(
		buttons[reason],
		payload,
	)
}

// makeCheckReceiptKeyboard - set wanna keyboard.
func makeCheckReceiptKeyboard(id string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			makeKeyboardButton(decisionAcceptGeneral, id, true),
			makeKeyboardButton(decisionAcceptCats, id, true),
		),
		tgbotapi.NewInlineKeyboardRow(
			makeKeyboardButton(decisionRejectAmountMismatch, id, false),
			makeKeyboardButton(decisionRejectTooOld, id, false),
			makeKeyboardButton(decisionRejectElectronic, id, false),
		),
		tgbotapi.NewInlineKeyboardRow(
			makeKeyboardButton(decisionRejectUnreadable, id, false),
			makeKeyboardButton(decisionRejectUnverifiable, id, false),
			makeKeyboardButton(decisionRejectIncomplete, id, false),
		),
		tgbotapi.NewInlineKeyboardRow(
			makeKeyboardButton(decisionRejectWithCallback, id, false),
			makeKeyboardButton(decisionRejectUnacceptable, id, false),
		),
	)
}
