package main

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
	decisionRejectBankCard
	decisionRejectElectronic
	decisionRejectIncomplete
	decisionRejectUnverifiable
	decisionRejectAmountMismatch
	decisionRejectTooOld
	decisionRejectWithCallback
	decisionRejectDoubled
	decisionRejectBusy
)

var buttons = map[int]string{
	decisionUnknown:              "\U0001f937 Не знаю",
	decisionAcceptGeneral:        "\U0001f44d Подтвердить",
	decisionAcceptCats:           "\U0001f63b Котики",
	decisionRejectUnacceptable:   "\U0001f595 Бан", //\U0000274c
	decisionRejectUnreadable:     "\U0001f576 Нечит.",
	decisionRejectBankCard:       "\U0001F4B3 Безнал",
	decisionRejectElectronic:     "\U0001f4f1 Эл-ый",
	decisionRejectIncomplete:     "\U0001f313 Неполн.",
	decisionRejectUnverifiable:   "\U0001f46e Непров.",
	decisionRejectAmountMismatch: "\U0001f4b5 Сумма",
	decisionRejectTooOld:         "\U0001f4c5 Старый",
	decisionRejectWithCallback:   "\u260e\ufe0f Свяжитесь",
	decisionRejectDoubled:        "\u270c\ufe0f Повтор",
	decisionRejectBusy:           "\U0001f4a3 Занят",
}

/*

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
			makeKeyboardButton(decisionRejectBankCard, id, false),
			makeKeyboardButton(decisionRejectUnreadable, id, false),
			makeKeyboardButton(decisionRejectUnverifiable, id, false),
		),
		tgbotapi.NewInlineKeyboardRow(
			makeKeyboardButton(decisionRejectIncomplete, id, false),
			makeKeyboardButton(decisionRejectWithCallback, id, false),
			makeKeyboardButton(decisionRejectUnacceptable, id, false),
		),
	)
}

*/
