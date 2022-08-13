package main

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

const (
	// MsgWelcome - welcome message.
	MsgWelcome = `Привет! 

Это VPN Generator - простой и бесплатный способ завести свой собственный VPN для друзей и родных. Нажми “Хочу свой VPN“, чтобы начать регистрацию.

У тебя уже есть VPN на наших мощностях и что-то не так? Нажми “Задать вопрос“ и мы ответим. Но не факт, что быстро:)`

	// MsgQuiz - quiz message.
	MsgQuiz = `Сейчас будет немного странное. Мы очень не хотим брать ни твой телефон, ни имейл на случай, если к нам придут злые дяди /в полосатых купальниках/ в форме, заберут эти данные. А потом как начнут их обогащать содержимым госуслуг и прочих утечек и будет грустно. Поэтому тебе придется:

Сходить ножками в магазин и что-нибудь купить на 500 рублей. Не нам, себе. Ну или своему котику. 
	
Заплатить за это наличкой. Прям бумажными деньгами - это нужно для твоей безопасности, чтобы процесс проверки никак не мог тебя деанонимизировать. И никаких карт лояльности!!!!

Прислать нам фотку чека с явно видным QR-кодом

Мы поймем, что ты живой нормальный человек, а не тролль испод моста и дадим тебе доступ к системе управления твоим собственным VPN-ом

Ждем фоточку чека!`

	// MsgAttestationAssigned - receipt have accepted.
	MsgAttestationAssigned = `Чек принят к рассмотрению`

	// WarnGroupsNotAllowed - this bot is only private.
	WarnGroupsNotAllowed = `Извини, в групповых чатах я не общаюсь`

	// WarnForbidForwards - this bot is only private.
	WarnForbidForwards = `Извини, я не работаю с пересылками`

	// WarnUnknownCommand - unknown command.
	WarnUnknownCommand = `Извини, я не знаком с этим указанием`

	// FatalUnwellSecurity - if autodelete not set.
	FatalUnwellSecurity = `Привет!

Установи автоудаление сообщений в этом чате *через 1 или 2 дня* и продолжи.`

	// WarnRequiredPhoto - warning about photo absents.
	WarnRequiredPhoto = `Ты забыл прикрепить фотографию чека.`

	// FatalSomeThingWrong - something wrong happened.
	FatalSomeThingWrong = `Мне жаль, что-то пошло не так`

	// DefaultSupportURL - support URL if isn't set.
	DefaultSupportURL = "https://t.me/"

	// ResetSuccessfull - Resety session.
	ResetSuccessfull = `Диалог сброшен`
)

var (
	// WannabeKeyboard - wanna keyboard.
	WannabeKeyboard tgbotapi.InlineKeyboardMarkup //nolint

	// StandartChatActions - something in status.
	StandartChatActions = [...]string{ //nolint
		"typing",
		"choose_sticker",
		"upload_photo",
		"record_video",
		"record_voice",
	}

	// ContinueKeyboard - continue keyboard.
	ContinueKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Продолжить", "continue"),
		),
	)
)

// SetWannaKeyboard - set wanna keyboard.
func SetWannaKeyboard(url string) {
	WannabeKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Хочу свой VPN", "wannabe"),
			tgbotapi.NewInlineKeyboardButtonURL("Задать вопрос", url),
		),
	)
}
