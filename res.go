package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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

Мы поймем, что ты живой нормальный человек, а не тролль из-под моста, и дадим тебе доступ к системе управления твоим собственным VPN-ом

Ждем фоточку чека!`

	// MsgAttestationAssigned - receipt have accepted.
	MsgAttestationAssigned = `Чек принят к рассмотрению`

	// WarnGroupsNotAllowed - this bot is only private.
	WarnGroupsNotAllowed = `Извини, в групповых чатах я не общаюсь`

	// WarnPrivateNotAllowed - this bot is only private.
	WarnPrivateNotAllowed = `Извини, в личках я не общаюсь`

	// WarnForbidForwards - this bot is only private.
	WarnForbidForwards = `Извини, в целях твоей же безопасности пересылка отключена`

	// WarnUnknownCommand - unknown command.
	WarnUnknownCommand = `Извини, но эта команда мне не знакома`

	// FatalUnwellSecurity - if autodelete not set.
	FatalUnwellSecurity = `Привет!

Установи пожалуйста автоудаление сообщений в этом чате на 1 день, если на твоем клиенте это возможно. [Инструкция](https://telegram.org/blog/autodelete-inv2/ru?ln=a)`

	// WarnRequiredPhoto - warning about photo absents.
	WarnRequiredPhoto = `Похоже ты забыл прикрепить фотографию чека.`

	// FatalSomeThingWrong - something wrong happened.
	FatalSomeThingWrong = `Похоже что-то пошло не так. Если ты уверен(-а), что все сделал(-а) правильно - напиши пожалуйста в [поддержку](%s)`

	// DefaultSupportURL - support URL if isn't set.
	DefaultSupportURL = "https://t.me/"

	// ResetSuccessfull - Resety session.
	ResetSuccessfull = `Диалог сброшен`

	// GrantMessage - you are brigadier.
	GrantMessage = `Поздравляю! Ты — бригадир!

    Последний, но важный шаг. У меня есть для тебя твое имя и 6 слов - их я дам. Их нужно где-то хранить - места для хранения я не дам. 

	Спрячь эти слова туда, куда ты сможешь добраться в любой непонятной ситуации, но не доберется трщ майор. Нет, не туда! Туда доберется :( Лучше - в харнилку паролей или еще какое-нибудь хитрое место.`

	// RejectMessage - we are shame you.
	RejectMessage = `Мне не понравился чек. Пришли другой пожалуйста.`
)

var (
	// WannabeKeyboard - wanna keyboard.
	WannabeKeyboard tgbotapi.InlineKeyboardMarkup //nolint
	// CheckBillKeyboard - check bill keyboard.
	CheckBillKeyboard tgbotapi.InlineKeyboardMarkup //nolint

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

	// FatalSomeThingWrongWithLink - fatal warning with support link.
	FatalSomeThingWrongWithLink string //nolint
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

// SetFatalSomeThingWrongWithLink - set link in fatal warning string.
func SetFatalSomeThingWrongWithLink(link string) {
	FatalSomeThingWrongWithLink = fmt.Sprintf(FatalSomeThingWrong, tgbotapi.EscapeText(tgbotapi.ModeMarkdown, link))
}
