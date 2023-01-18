package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (

	// Support additional
	extraSupportText = "Если ты уверен(-а), что все сделал(-а) правильно - напиши пожалуйста в [поддержку](%s)."

	// MainTrackUnwellSecurityMessage - if autodelete not set.
	MainTrackUnwellSecurityMessage = `Привет!

Установи пожалуйста автоудаление сообщений в этом чате на 1 день, если на твоем клиенте это возможно. [Инструкция](https://telegram.org/blog/autodelete-inv2/ru?ln=a)`

	// MainTrackWelcomeMessage - welcome message.
	MainTrackWelcomeMessage = `Привет! 

Это VPN Generator — простой и *бесплатный* способ завести свой собственный VPN для друзей и родных. Нажми «Хочу свой VPN», чтобы начать регистрацию.

После регистрации ты станешь бригадиром и получишь в свои руки _ключницу_ — инструмент, который будет генерировать VPN-конфигурации для тех, с кем ты захочешь ими поделиться.

У тебя уже есть VPN на наших мощностях и что-то не так? Нажми «Задать вопрос» и мы ответим... Но не факт, что быстро ` + "\U0000263A." +
		`

VPN Generator находится на начальном этапе своего развития. Поэтому пока что VPN Generator работает на базе безопасного решения [Wireguard](https://www.wireguard.com/) с открытым исходным кодом. В дальнейшем мы будем добавлять другие протоколы и, при необходимости, реализуем свой.
`

	// mainTrackQuizMessage - quiz message.
	mainTrackQuizMessage = `Сейчас будет немного странное. Мы очень не хотим брать ни твой телефон, ни имейл на случай, если к нам придут злые дяди в форме и заберут эти данные. А потом как начнут их обогащать содержимым госуслуг и прочих утечек, и будет грустно. 

Но нам нужно понять, что ты живой нормальный человек, а не тролль из-под моста. Поэтому тебе придется:
	
• Сходить ножками в магазин и что-нибудь купить примерно на 500 рублей или больше, ровно-ровно набивать не нужно. Купить не нам, себе. Ну или своему котику. 
	
• Заплатить за это *наличкой*. Прям бумажными деньгами — это нужно для твоей безопасности, чтобы процесс проверки никак не мог тебя деанонимизировать. *И никаких карт лояльности*!!!!
	
• Прислать нам фотку чека не старше 7 дней. Если чек из РФ, то на нём должен быть хорошо различимый проверяемый QR-код налоговой.

Ждем фоточку чека и мы дадим тебе твой VPN!

_Если у тебя есть вопросы, почему твой чек отклонен — напиши в_ [поддержку](%s), _мы расскажем_ ` + "\U0000263A."

	// MainTrackSendForAttestationMessage - receipt have accepted.
	MainTrackSendForAttestationMessage = `Чек принят к рассмотрению.`

	// mainTrackWarnRequiredPhoto - warning about photo absents.
	mainTrackWarnRequiredPhoto = `Похоже ты забыл прикрепить фотографию чека. Попробуй ещё раз. Если ты не помнишь условий, используй команду /repeat . ` + extraSupportText

	// mainTrackWarnWaitForApprovement
	mainTrackWarnWaitForApprovement = `Ожидай подтверждения чека. Это может занять какое-то время. Если у тебя остались вопросы - напиши пожалуйста в [поддержку](%s).`

	// MainTrackGrantMessage - grant message.
	MainTrackGrantMessage = "Поздравляю! Ты — бригадир!\nТвое кодовое имя: `%s`. Оно нужно для обращения в поддержку. Так мы поймем, что ты — это ты, не зная, что это ты \U0000263A."

	// MainTrackPersonDescriptionMessage - brief on name.
	MainTrackPersonDescriptionMessage = "*Справка*\n\nЛауреат нобелевской премии по физике: *%s*\n_%s_\n%s\n\n"

	// MainTrackConfigFormatFileCaption - config file caption.
	MainTrackConfigFormatFileCaption = "Конфигурация файлом"

	// MainTrackConfigFormatTextTemplate - config text template.
	MainTrackConfigFormatTextTemplate = "Конфигурация текстом:\n```\n%s```"

	// MainTrackConfigFormatQRCaption - qr-config caption.
	MainTrackConfigFormatQRCaption = "Конфигурация QR-кодом"

	// MainTrackSeedDescMessage - you are brigadier.
	MainTrackSeedDescMessage = `Последний, но важный шаг. У меня есть для тебя 6 слов — их я дам. Их нужно где-то хранить — места для хранения я не дам. Эти слова + имя — единственный способ восстановить доступ к твоему VPN.

Спрячь эти слова туда, куда ты сможешь добраться в любой непонятной ситуации, но не доберется трщ майор. Нет, не туда! Туда доберется… Лучше в хранилку паролей или еще какое-нибудь хитрое место.`

	// MainTrackWordsMessage - 6 words.
	MainTrackWordsMessage = "6 важных слов:\n`%s`"

	// MainTrackConfigsMessage - keydesk address and config.
	MainTrackConfigsMessage = `Ниже ты найдешь адрес твоей ключницы и файл конфигурации. Добавь конфигурацию в Wireguard на устройстве, с которого будешь потом управлять VPN-ом и обязательно зайди в ключницу по ссылке [http://vpn.works/](http://vpn.works/) (*работает только с подключенным VPN!*). Следуй инструкции и оставайся на связи!`

	// MainTrackKeydeskIPv6Message - message with ipv6 keydesk address.
	MainTrackKeydeskIPv6Message = "\U0001f510 " + `Возможно ты тоже энтузиаст(-ка) безопасности и у тебя установлен защищённый DNS в системе или в браузере. Тогда ссылка на ключницу не будет работать, потому что она существует только в нашем DNS. Безопасность требует жертв и тебе придётся в ключницу напрямую по IPv6-адресу: ` + "`http://[%s]/`" +

		`

P.S. К сожалению, этот способ не будет работать в мобильной версии браузера Firefox`

	// mainTrackWarnConversationsFinished - dialog after the end.
	mainTrackWarnConversationsFinished = `Наш приятный разговор завершён. 
	
Если ты забыл(-а) своё имя и 6 важных слов, то даже в поддержке мы ничем помочь не сможем. Это условие нашей безопасности.
	
` + extraSupportText

	// MainTrackResetSuccessfull - Resety session.
	MainTrackResetSuccessfull = `Диалог сброшен`

	// mainTrackFailMessage - something wrong during creation time.
	mainTrackFailMessage = `Что-то сломалось. Попробуй ещё раз позже. Если не получится, напиши в наш бот поддержки: %s или на электропочту: %s.`

	// InfoGroupsNotAllowedMessage - this bot is only private.
	InfoGroupsNotAllowedMessage = `Извини, в групповых чатах я не общаюсь.`

	// InfoPrivateNotAllowedMessage - this bot is only private.
	InfoPrivateNotAllowedMessage = `Извини, в личках я не общаюсь.`

	// InfoForbidForwardsMessage - this bot is only private.
	InfoForbidForwardsMessage = `Извини, в целях твоей же безопасности пересылка отключена.`

	// infoUnknownCommandMessage - unknown command.
	infoUnknownCommandMessage = `Извини, но эта команда мне не знакома. ` + extraSupportText

	// fatalSomeThingWrong - something wrong happened.
	fatalSomeThingWrong = `Похоже что-то пошло не так. ` + extraSupportText

	// DefaultSupportURLText - support URL if isn't set.
	DefaultSupportURLText = "https://t.me/"

	// rejectMessage - we are shame you.
	rejectMessage = `Что-то пошло не так. Попробуй пожалуйста перефоткать чек и проверить, соответствует ли то, что ты прислал, условиям выше. ` + extraSupportText
)

var (
	// MainTrackFailMessage - something wrong during creation time.
	MainTrackFailMessage string

	// MainTrackQuizMessage - quiz message.
	MainTrackQuizMessage string

	// WannabeKeyboard - wanna keyboard.
	WannabeKeyboard tgbotapi.InlineKeyboardMarkup //nolint
	// CheckBillKeyboard - check bill keyboard.
	CheckBillKeyboard tgbotapi.InlineKeyboardMarkup //nolint

	// StandardChatActions - something in status.
	StandardChatActions = [...]string{ //nolint
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
	// InfoUnknownCommandMessage - unknown command.
	InfoUnknownCommandMessage string
	// MainTrackWarnRequiredPhoto -.
	MainTrackWarnRequiredPhoto string
	// MainTrackWarnWaitForApprovement - .
	MainTrackWarnWaitForApprovement string
	// MainTrackWarnConversationsFinished - dialog after the end.
	MainTrackWarnConversationsFinished string

	// RejectMessage - we are shame you.
	RejectMessage string

	// DecisionComments - descriptive text on check decidion.
	DecisionComments = map[int]string{
		decisionUnknown:              "",
		decisionAcceptGeneral:        "",
		decisionAcceptCats:           "",
		decisionRejectUnacceptable:   "",
		decisionRejectUnreadable:     "",
		decisionRejectBankCard:       "",
		decisionRejectElectronic:     "",
		decisionRejectIncomplete:     "",
		decisionRejectUnverifiable:   "",
		decisionRejectAmountMismatch: "",
		decisionRejectTooOld:         "",
		decisionRejectWithCallback:   "",
	}

	// decisionCommentsTemplate - descriptive text on check decidion.
	decisionCommentsTemplate = map[int]string{
		decisionUnknown:              "Что-то пошло не так... " + extraSupportText,
		decisionAcceptCats:           "Котики - это святое! \U0001f63b",
		decisionRejectUnacceptable:   "Похоже, ты прислал(-а) что-то очень нехорошее. Тебя забанили в сервисе на веки \U0001f937 . " + extraSupportText,
		decisionRejectUnreadable:     "Пожалуйста, пришли читаемый чек! " + extraSupportText,
		decisionRejectBankCard:       "Похоже, ты оплатил(-а) картой. Пожалуйста принеси чек, оплаченный наличкой. " + extraSupportText,
		decisionRejectElectronic:     "Пожалуйста, пришли сам чек, а не результат его расшифровки! " + extraSupportText,
		decisionRejectIncomplete:     "Пожалуйста, пришли чек целиком! " + extraSupportText,
		decisionRejectUnverifiable:   "Чек не бьется с налоговой, пришли пожалуйста другой чек! " + extraSupportText,
		decisionRejectAmountMismatch: "Похоже что-то серьезно не так с суммой чека. Пришли пожалуйста другой! " + extraSupportText,
		decisionRejectTooOld:         "Похоже чек устарел. Пришли пожалуйста тот, что не старше недели. " + extraSupportText,
		decisionRejectWithCallback:   "Похоже что-то не так с чеком и нам нужно поговорить. Свяжись пожалуйста с [нами](%s).",
	}
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
	MainTrackFailMessage = fmt.Sprintf(mainTrackFailMessage, link, email)
	MainTrackQuizMessage = fmt.Sprintf(mainTrackQuizMessage, link)
	FatalSomeThingWrong = fmt.Sprintf(fatalSomeThingWrong, link)
	InfoUnknownCommandMessage = fmt.Sprintf(infoUnknownCommandMessage, link)
	RejectMessage = fmt.Sprintf(rejectMessage, link)
	MainTrackWarnRequiredPhoto = fmt.Sprintf(mainTrackWarnRequiredPhoto, link)
	MainTrackWarnWaitForApprovement = fmt.Sprintf(mainTrackWarnWaitForApprovement, link)
	MainTrackWarnConversationsFinished = fmt.Sprintf(mainTrackWarnConversationsFinished, link)

	DecisionComments[decisionUnknown] = fmt.Sprintf(decisionCommentsTemplate[decisionUnknown], link)
	DecisionComments[decisionAcceptCats] = decisionCommentsTemplate[decisionAcceptCats]
	DecisionComments[decisionRejectUnacceptable] = fmt.Sprintf(decisionCommentsTemplate[decisionRejectUnacceptable], link)
	DecisionComments[decisionRejectUnreadable] = fmt.Sprintf(decisionCommentsTemplate[decisionRejectUnreadable], link)
	DecisionComments[decisionRejectBankCard] = fmt.Sprintf(decisionCommentsTemplate[decisionRejectBankCard], link)
	DecisionComments[decisionRejectElectronic] = fmt.Sprintf(decisionCommentsTemplate[decisionRejectElectronic], link)
	DecisionComments[decisionRejectIncomplete] = fmt.Sprintf(decisionCommentsTemplate[decisionRejectIncomplete], link)
	DecisionComments[decisionRejectUnverifiable] = fmt.Sprintf(decisionCommentsTemplate[decisionRejectUnverifiable], link)
	DecisionComments[decisionRejectAmountMismatch] = fmt.Sprintf(decisionCommentsTemplate[decisionRejectAmountMismatch], link)
	DecisionComments[decisionRejectTooOld] = fmt.Sprintf(decisionCommentsTemplate[decisionRejectTooOld], link)
	DecisionComments[decisionRejectWithCallback] = fmt.Sprintf(decisionCommentsTemplate[decisionRejectWithCallback], link)
}
