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

	// WarnGroupsNotAllowed - this bot is only private.
	WarnGroupsNotAllowed = `Извините, в группах бот не работает`
	// FatalUnwellSecurity - if autodelete not set
	FatalUnwellSecurity = `Привет!

Установи автоудаление сообщений в этом чате *через 1 день* и продолжи.`

	// FatalSomeThingWrong - something wrong happened
	FatalSomeThingWrong = `Что-то пошло не так`
)

var wannabeKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Хочу свой VPN", "wannabe"),
		tgbotapi.NewInlineKeyboardButtonURL("Задать вопрос", "https://t.me/durov"),
	),
)
