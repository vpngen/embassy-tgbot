package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (

	// Support additional
	extraSupport = "Если ты уверен(-а), что все сделал(-а) правильно - напиши пожалуйста в [поддержку](%s)."

	// MsgWelcome - welcome message.
	MsgWelcome = `Привет! 

Это VPN Generator — простой и *бесплатный* способ завести свой собственный VPN для друзей и родных. Нажми «Хочу свой VPN», чтобы начать регистрацию.

После регистрации ты станешь бригадиром и получишь в свои руки _ключницу_ — инструмент, который будет генерировать VPN-конфигурации для тех, с кем ты захочешь ими поделиться.

У тебя уже есть VPN на наших мощностях и что-то не так? Нажми «Задать вопрос» и мы ответим... Но не факт, что быстро ` + "\U0000263A." +
		`

VPN Generator находится на начальном этапе своего развития. Поэтому пока что VPN Generator работает на базе безопасного решения [Wireguard](https://www.wireguard.com/) с открытым исходным кодом. В дальнейшем мы будем добавлять другие протоколы и, при необходимости, реализуем свой.
`

	// msgQuiz - quiz message.
	msgQuiz = `Сейчас будет немного странное. Мы очень не хотим брать ни твой телефон, ни имейл на случай, если к нам придут злые дяди в форме и заберут эти данные. А потом как начнут их обогащать содержимым госуслуг и прочих утечек, и будет грустно. 

Но нам нужно понять, что ты живой нормальный человек, а не тролль из-под моста. Поэтому тебе придется:
	
• Сходить ножками в магазин и что-нибудь купить примерно на 500 рублей или больше, ровно-ровно набивать не нужно. Купить не нам, себе. Ну или своему котику. 
	
• Заплатить за это *наличкой*. Прям бумажными деньгами — это нужно для твоей безопасности, чтобы процесс проверки никак не мог тебя деанонимизировать. *И никаких карт лояльности*!!!!
	
• Прислать нам фотку чека не старше 7 дней. Если чек из РФ, то на нём должен быть хорошо различимый проверяемый QR-код налоговой.

Ждем фоточку чека и мы дадим тебе твой VPN!

_Если у тебя есть вопросы, почему твой чек отклонен — напиши в_ [поддержку](%s), _мы расскажем_ ` + "\U0000263A."

	// MsgAttestationAssigned - receipt have accepted.
	MsgAttestationAssigned = `Чек принят к рассмотрению.`

	// WarnGroupsNotAllowed - this bot is only private.
	WarnGroupsNotAllowed = `Извини, в групповых чатах я не общаюсь.`

	// WarnPrivateNotAllowed - this bot is only private.
	WarnPrivateNotAllowed = `Извини, в личках я не общаюсь.`

	// WarnForbidForwards - this bot is only private.
	WarnForbidForwards = `Извини, в целях твоей же безопасности пересылка отключена.`

	// warnUnknownCommand - unknown command.
	warnUnknownCommand = `Извини, но эта команда мне не знакома. ` + extraSupport

	// FatalUnwellSecurity - if autodelete not set.
	FatalUnwellSecurity = `Привет!

Установи пожалуйста автоудаление сообщений в этом чате на 1 день, если на твоем клиенте это возможно. [Инструкция](https://telegram.org/blog/autodelete-inv2/ru?ln=a)`

	// warnRequiredPhoto - warning about photo absents.
	warnRequiredPhoto = `Похоже ты забыл прикрепить фотографию чека. Попробуй ещё раз. Если ты не помнишь условий, используй команду /repeat . ` + extraSupport

	// warnWaitForApprovement
	warnWaitForApprovement = `Ожидай подтверждения чека. Это может занять какое-то время. Если у тебя остались вопросы - напиши пожалуйста в [поддержку](%s).`

	// fatalSomeThingWrong - something wrong happened.
	fatalSomeThingWrong = `Похоже что-то пошло не так. ` + extraSupport

	// DefaultSupportURL - support URL if isn't set.
	DefaultSupportURL = "https://t.me/"

	// ResetSuccessfull - Resety session.
	ResetSuccessfull = `Диалог сброшен`

	// GrantMessage - grant message.
	GrantMessage = "Поздравляю! Ты — бригадир!\nТвое кодовое имя: `%s`. Оно нужно для обращения в поддержку. Так мы поймем, что ты — это ты, не зная, что это ты \U0000263A."

	// rejectMessage - we are shame you.
	rejectMessage = `Что-то пошло не так. Попробуй пожалуйста перефоткать чек и проверить, соответствует ли то, что ты прислал, условиям выше. ` + extraSupport
	// SeedMessage - you are brigadier.
	SeedMessage = `Последний, но важный шаг. У меня есть для тебя 6 слов — их я дам. Их нужно где-то хранить — места для хранения я не дам. Эти слова + имя — единственный способ восстановить доступ к твоему VPN.

Спрячь эти слова туда, куда ты сможешь добраться в любой непонятной ситуации, но не доберется трщ майор. Нет, не туда! Туда доберется… Лучше в хранилку паролей или еще какое-нибудь хитрое место.`

	// WordsMessage - 6 words.
	WordsMessage = "6 важных слов:\n`%s`"

	// FinalMessage - keydesk address and config.
	FinalMessage = `Ниже ты найдешь адрес твоей ключницы и файл конфигурации. Добавь конфигурацию в Wireguard на устройстве, с которого будешь потом управлять VPN-ом и обязательно зайди в ключницу по ссылке [http://vpn.works/](http://vpn.works/) (*работает только с подключенным VPN!*). Следуй инструкции и оставайся на связи!`

	// KeydeskIPv6Message - message with ipv6 keydesk address.
	KeydeskIPv6Message = "\U0001f510 " + `Возможно ты тоже энтузиаст(-ка) безопасности и у тебя установлен защищённый DNS в системе или в браузере. Тогда ссылка на ключницу не будет работать, потому что она существует только в нашем DNS. Безопасность требует жертв и тебе придётся в ключницу напрямую по IPv6-адресу: ` + "`http://[%s]/`" +

		`

P.S. К сожалению, этот способ не будет работать в мобильной версии браузера Firefox`

	// failMessage - something wrong during creation time.
	failMessage = `Что-то сломалось. Попробуй ещё раз позже. Если не получится, напиши в наш бот поддержки: %s или на электропочту: %s.`

	// warnConversationsFinished - dialog after the end.
	warnConversationsFinished = `Наш приятный разговор завершён. 
	
Если ты забыл(-а) своё имя и 6 важных слов, то даже в поддержке мы ничем помочь не сможем. Это условие нашей безопасности.
	
` + extraSupport
)

var (
	// FailMessage - something wrong during creation time.
	FailMessage string

	// MsgQuiz - quiz message.
	MsgQuiz string

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

	// FatalSomeThingWrong - fatal warning with support link.
	FatalSomeThingWrong string
	// WarnUnknownCommand - unknown command.
	WarnUnknownCommand string
	// RejectMessage - we are shame you.
	RejectMessage string
	// WarnRequiredPhoto -.
	WarnRequiredPhoto string
	// WarnWaitForApprovement - .
	WarnWaitForApprovement string
	// WarnConversationsFinished - dialog after the end.
	WarnConversationsFinished string
)

// SetSupportMessages - set wanna keyboard.
func SetSupportMessages(url, email string) {
	link := tgbotapi.EscapeText(tgbotapi.ModeMarkdown, url)

	WannabeKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Хочу свой VPN", "started"),
			tgbotapi.NewInlineKeyboardButtonURL("Задать вопрос", url),
		),
	)
	FailMessage = fmt.Sprintf(failMessage, link, email)
	MsgQuiz = fmt.Sprintf(msgQuiz, link)
	FatalSomeThingWrong = fmt.Sprintf(fatalSomeThingWrong, link)
	WarnUnknownCommand = fmt.Sprintf(WarnUnknownCommand, link)
	RejectMessage = fmt.Sprintf(rejectMessage, link)
	WarnRequiredPhoto = fmt.Sprintf(warnRequiredPhoto, link)
	WarnWaitForApprovement = fmt.Sprintf(warnWaitForApprovement, link)
	WarnConversationsFinished = fmt.Sprintf(warnConversationsFinished, link)
}
