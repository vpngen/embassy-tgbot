package main

import (
	"fmt"

	_ "embed"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:embed vgbs.png
var RestoreTrackImgVgbs []byte

const (

	// Support additional
	extraSupportText      = "Если ты уверен(-а), что все сделал(-а) правильно — напиши пожалуйста в [поддержку](%s)."
	extraSupportTextShort = "Или напиши в [поддержку](%s)."

	RestoreTrackStartMessage = `Здесь восстанавливаются бригады. Мы сможем *восстановить твою бригаду* если у тебя потерян конфиг и тебе нужно восстановить контроль над бригадой. 

Или
	
Мы сможем выдать тебе *новую бригаду* в случае, если мы удалили твою старую за нарушение условий использования. Новая бригада означает, что тебе придется раздавать конфиги пользователям заново.
	
В обоих случаях _обязательно знать  имя бригадира и шесть волшебных слов_. Поехали?
`

	RestoreTrackNameMessage  = `Чтобы восстановить свой конфиг, напиши пожалуйста имя бригадира. Т.е. прилагательное и фамилию лауреата, например “Веселый Эйнштейн” или “Потрясающая Кюри”.`
	RestoreTrackWordsMessage = `Супер, а теперь назови пожалуйста 6 волшебных слов.`

	RestoreTrackInvalidNameMessage = `Это не похоже на имя, попробуй еще раз. Будь внимателен(-на), тебе нужно ввести прилагательное и фамилию лауреата, например “Веселый Эйнштейн” или “Потрясающая Кюри”.  

Если ты не помнишь имя — возможно стоит начать сначала и идти за новой картинкой.`

	RestoreTrackBrigadeNotFoundMessage = `Сим-сим не открылся, с именем или словами что-то не так. Будь внимателен(-на), имя состоит из прилагательного и фамилии лауреата, например “Веселый Эйнштейн” или “Потрясающая Кюри”. А шесть слов должны быть ровно в той последовательности, в которой мы тебе их отдали!

Если ты не помнишь имя и/или волшебные слова — возможно стоит начать сначала и идти за новой картинкой.

А еще вспомни, когда ты в последний раз пользовался нашим VPN? Если это было давно - мы тебя удалили и пора идти за новой картинкой.`

	RestoreTrackGrantMessage = `Мы узнали тебя, держи свой конфиг!`

	RestoreTracIP2DomainHintsMessage = `У тебя есть 2 способа восстановить конфиги членов своей бригады:

• Выпустить всем членам бригады новые конфиги (подойдет для небольших бригад)
• Попросить членов бригады заменить в последней строчку строчке конфигурации WireGuard _Endpoint_ IP-адрес (набор чисел, разделенных точками, например ` + "`185.135.141.11`" + `) на ` + "`%s`" + `. И все заработает!`

	// MainTrackUnwellSecurityMessage - if autodelete not set.
	MainTrackUnwellSecurityMessage = `Привет!

Если ты в России/Беларуси и беспокоишься о своей безопасности — пожалуйста, установи автоудаление сообщений в этом чате на 1 день. [Инструкция](https://telegram.org/blog/autodelete-inv2/ru?ln=a)`

	// MainTrackWelcomeMessage - welcome message.
	MainTrackWelcomeMessage = `Привет! 

Это VPN Generator — простой и *бесплатный* способ завести свой собственный VPN для друзей и родных. Нажми «Хочу свой VPN», чтобы начать регистрацию.

ВАЖНО: здесь ты получишь целый сервер. Это ценный и очень ограниченный ресурс, который ты получишь бесплатно. Поэтому тебе придется соответствовать простым требованиям:
• Начать пользоваться системой управления сервером в течение 24 часов.
• Начиная со следующего месяца после твоей регистрации иметь не менее пяти активных пользователей в месяц. Т.е. начать распространять VPN в своем окружении.

Или мы удалим твою бригаду ` + "\U0001F937" + `.

У тебя уже есть VPN на наших мощностях и что-то не так? Нажми «Задать вопрос» и мы ответим... Но не факт, что быстро ` + "\U0000263A." + `

Чтобы узнавать о возможностях, проблемах и в целом о развитии проекта, обязательно подпишись на [@vpngen](https://t.me/vpngen). Инструкция как быть бригадиром - [тут](https://docs.google.com/document/d/12qFYFk9SQaPrg32bf-2JZYIPSax2453jE3YGOblThHk/)
`

	// mainTrackQuizMessage_zero - quiz message.
	/* mainTrackQuizMessage_zero = `Сейчас будет немного странное. Мы очень не хотим брать ни твой телефон, ни имейл на случай, если к нам придут злые дяди в форме и заберут эти данные. А потом как начнут их обогащать содержимым госуслуг и прочих утечек, и будет грустно.

	Но нам нужно понять, что ты живой нормальный человек, а не тролль из-под моста. Поэтому тебе придется:

	• Сходить ножками в магазин и что-нибудь купить примерно на 500 рублей (или эквивалент в твоей валюте) или больше, ровно-ровно набивать не нужно. Купить не нам, себе. Ну или своему котику.

	• Заплатить за это *наличкой*. Прям бумажными деньгами — это нужно для твоей безопасности, чтобы процесс проверки никак не мог тебя деанонимизировать. *И никаких карт лояльности*!!!!

	• Прислать нам фотку чека с хорошо читаемым QR-кодом, если ты в России. Если не в России — QR код рисовать не надо.

	Ждем фоточку чека и мы дадим тебе твой VPN!

	_Если у тебя есть вопросы, почему твой чек отклонен — напиши в_ [поддержку](%s), _мы расскажем_ ` + "\U0000263A." */

	// mainTrackQuizMessage_one
	mainTrackQuizMessage_one = `*Круто, что ты решил воспользоваться VPN Generator! Он у нас бесплатный, и  нам не нужны твои телефон и емайл.*

Но мы хотим понять, что ты живой нормальный человек, а не троль из-под моста, поэтому пришли, пожалуйста, *любую картинку в этот чат* и жди наших инструкций.
	
_Если у тебя появятся вопросы — напиши нам в_ [поддержку](%s), _мы расскажем_ ` + "\U0000263A."

	// MainTrackSendForAttestationMessage - receipt have accepted.
	MainTrackSendForAttestationMessage = `Воу, какая картинка! Вся наша команда рассматривает ее и скоро ответит тебе.`

	// mainTrackWarnRequiredPhoto - warning about photo absents.
	mainTrackWarnRequiredPhoto = `Похоже ты забыл прикрепить картинку. Попробуй ещё раз. Если ты не помнишь условий, используй команду /repeat . ` + extraSupportText

	// mainTrackWarnWaitForApprovement
	mainTrackWarnWaitForApprovement = `Ожидай пока вся наша команда посмотрит на твою картинку. Это может занять какое-то время. Если у тебя остались вопросы — напиши пожалуйста в [поддержку](%s).`

	// MainTrackGrantMessage - grant message.
	MainTrackGrantMessage = "Поздравляю! Ты — бригадир! Вот полная [инструкция пользования](https://docs.google.com/document/d/12qFYFk9SQaPrg32bf-2JZYIPSax2453jE3YGOblThHk/) сервисом.\nТвое кодовое имя: `%s`. Оно нужно для обращения в поддержку. Так мы поймем, что ты — это ты, не зная, что это ты \U0000263A."

	// MainTrackPersonDescriptionMessage - brief on name.
	MainTrackPersonDescriptionMessage = "*Справка*\n\nЛауреат нобелевской премии по физике: *%s*\n_%s_\n\U0001F449 [статья на wikipedia](%s)\n\n"

	// MainTrackConfigFormatFileCaption - config file caption.
	MainTrackConfigFormatFileCaption = "Твоя *личная* конфигурация Wireguard файлом"
	// mainTrackAmneziaOvcConfigFormatFileCaption - config file caption.
	mainTrackAmneziaOvcConfigFormatFileCaption = "На всякий случай твоя *личная* конфигурация AmneziaVPN файлом. Используй её только в случае, если ничего не работает. Если не работает и она — пиши в [поддержку](%s)."

	// MainTrackConfigFormatTextTemplate - config text template.
	MainTrackConfigFormatTextTemplate = "Твоя *личная* конфигурация Wireguard текстом:\n```\n%s```"

	// MainTrackOutlineAccessMessage - config text template.
	MainTrackOutlineAccessMessage = "*Шаг 2.* Твой *личный* ключ Outline ниже. Скопируй его, открой Outline и вставь. Затем обязательно нажми кнопку «Подключить»"

	// MainTrackIPSecL2TPManualConfigTemplate - config text template.
	MainTrackIPSecL2TPManualConfigTemplate = "Твоя *личная* конфигурация IPSec/L2TP:\nPreshared Key: `%s`\nUsername: `%s`\nPassword: `%s`\nServer: `%s`"

	// MainTrackConfigFormatQRCaption - qr-config caption.
	MainTrackConfigFormatQRCaption = "Твоя *личная* конфигурация Wireguard QR-кодом"

	// MainTrackSeedDescMessage - you are brigadier.
	MainTrackSeedDescMessage = `Последний, но важный шаг. У меня есть для тебя 6 слов — их я дам. Их нужно где-то хранить — места для хранения я не дам. Эти слова + имя — *единственный способ* восстановить доступ к твоему VPN.

Спрячь эти слова туда, куда ты сможешь добраться в любой непонятной ситуации, но не доберется трщ майор. Нет, не туда! Туда доберется… Лучше в хранилку паролей или еще какое-нибудь хитрое место.`

	// MainTrackWordsMessage - 6 words.
	MainTrackWordsMessage = "*6 важных слов:*\n`%s`"

	// MainTrackConfigsMessage - keydesk address and config.
	/* // subtask: VG-1581
		MainTrackConfigsMessage = `Выше — файл твоей *личной* конфигурации. Добавь конфигурацию в Wireguard на устройстве, с которого будешь потом управлять VPN-ом и *обязательно* зайди в ключницу *с включённым VPN* по ссылке [http://vpn.works/](http://vpn.works/) или напрямую по IPv6-адресу: ` + "`http://[%s]/`" + ` *в течение 24 часов* для активации бригады.

	P.S. Ты можешь закинуть свой конфиг в сохраненки, это достаточно безопасно.`
	*/

	//MainTrackConfigsMessage = `Выше — твои *личные* конфигурации VPN разных технологий. Выбери конфигурацию и добавь её на устройстве, с которого будешь потом управлять VPN-ом и *обязательно* зайди в ключницу *с включённым VPN* по ссылке [http://vpn.works/](http://vpn.works/) *в течение 24 часов* для активации бригады.
	//
	// P.S. Ты можешь закинуть свой конфиг в сохраненки, это достаточно безопасно.`

	MainTrackConfigsMessage = `*Шаг 3.* Подключился? Тогда *обязательно* перейди по ссылке [http://vpn.works/](http://vpn.works/) *в течение 24 часов*, чтобы активировать VPN и свою бригаду.`

	// MainTrackKeydeskIPv6Message - message with ipv6 keydesk address.
	MainTrackKeydeskIPv6Message = "\U0001f510 " + `Возможно ты тоже энтузиаст(-ка) безопасности и у тебя установлен защищённый DNS в системе или в браузере. Тогда ссылка на ключницу не будет работать, потому что она существует только в нашем DNS. Безопасность требует жертв и тебе придётся в ключницу напрямую по IPv6-адресу: ` + "`http://[%s]/`" +

		`

P.S. К сожалению, этот способ не будет работать в мобильной версии браузера Firefox`

	// mainTrackWarnConversationsFinished - dialog after the end.
	mainTrackWarnConversationsFinished = `Наш приятный разговор завершён. 
	
Если ты забыл(-а) своё имя и 6 важных слов, то даже в поддержке мы ничем помочь не сможем. Это условие нашей безопасности.
	
` + extraSupportText

	// repeatTrackWarnConversationsFinished - answer to /repeat after the end.
	repeatTrackWarnConversationsFinished = `Мы не можем повторить выданные данные, потому что мы их не храним. Если у тебя сохранились имя бригадира и 6 волшебных слов — ты можешь восстановить свою конфигурацию через [поддержку](%s).
	
Если и конфигурация, и слова для восстановления потеряны, боюсь, мы бессильны тебе помочь.`

	// MainTrackResetSuccessfull - Resety session.
	MainTrackResetSuccessfull = `Диалог сброшен`

	// mainTrackFailMessage - something wrong during creation time.
	mainTrackFailMessage = `Что-то сломалось. Попробуй ещё раз отослать картинку позже. ` + extraSupportText

	// InfoGroupsNotAllowedMessage - this bot is only private.
	InfoGroupsNotAllowedMessage = `Извини, в групповых чатах я не общаюсь.`

	// InfoPrivateNotAllowedMessage - this bot is only private.
	InfoPrivateNotAllowedMessage = `Извини, в личках я не общаюсь.`

	// InfoForbidForwardsMessage - this bot is only private.
	InfoForbidForwardsMessage = `Извини, в целях твоей же безопасности пересылка отключена.`

	// infoUnknownCommandMessage - unknown command.
	infoUnknownCommandMessage = `Если ты забыл(-а) на каком ты этапе, нажми /repeat . ` + extraSupportText

	// fatalSomeThingWrong - something wrong happened.
	fatalSomeThingWrong = `Похоже что-то пошло не так. ` + extraSupportText

	// DefaultSupportURLText - support URL if isn't set.
	DefaultSupportURLText = "https://t.me/"

	// rejectMessage - we are shame you.
	rejectMessage = `Что-то пошло не так. Попробуй пожалуйста перефоткать чек и проверить, соответствует ли то, что ты прислал, условиям выше. ` + extraSupportText
)

var (
	// RestoreWordsKeyboard1 - restore keyboard for words warn.
	RestoreWordsKeyboard1 tgbotapi.InlineKeyboardMarkup //nolint

	// RestoreWordsKeyboard2 - restore keyboard for words warn.
	RestoreWordsKeyboard2 tgbotapi.InlineKeyboardMarkup //nolint

	// RestoreNameKeyboard - restore keyboard for name warn.
	RestoreNameKeyboard tgbotapi.InlineKeyboardMarkup //nolint

	// RestoreStartKeyboard - restore keyboard.
	RestoreStartKeyboard tgbotapi.InlineKeyboardMarkup //nolint

	// MainTrackFailMessage - something wrong during creation time.
	MainTrackFailMessage string

	// MainTrackQuizMessage - quiz message.
	MainTrackQuizMessage map[string]string = map[string]string{} //nolint

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
	// RepeatTrackWarnConversationsFinished - answer to /repeat after the end.
	RepeatTrackWarnConversationsFinished string
	// MainTrackAmneziaOvcConfigFormatFileCaption - config file caption.
	MainTrackAmneziaOvcConfigFormatFileCaption string

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
		decisionRejectDoubled:        "",
		decisionRejectBusy:           "",
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
		decisionRejectDoubled:        "Похоже такую картинку нам уже присылали. Пришли пожалуйста другую. " + extraSupportText,
		decisionRejectBusy:           "Прости нас, очень много работы. Попробуй пожалуйста позже 🙏",
	}
)

const newTrack = "Пойду за картинкой"

// SetSupportMessages - set wanna keyboard.
func SetSupportMessages(url string) {
	link := tgbotapi.EscapeText(tgbotapi.ModeMarkdown, url)

	RestoreWordsKeyboard1 = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Попробовать ещё раз", "return"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(newTrack, "reset"),
			tgbotapi.NewInlineKeyboardButtonURL("Задать вопрос", url),
		),
	)

	RestoreWordsKeyboard2 = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Попробовать ещё раз", "again"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(newTrack, "reset"),
			tgbotapi.NewInlineKeyboardButtonURL("Задать вопрос", url),
		),
	)

	RestoreNameKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(newTrack, "reset"),
			tgbotapi.NewInlineKeyboardButtonURL("Задать вопрос", url),
		),
	)

	RestoreStartKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Начать", "restore"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(newTrack, "reset"),
			tgbotapi.NewInlineKeyboardButtonURL("Задать вопрос", url),
		),
	)

	WannabeKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Хочу свой VPN", "started"),
			tgbotapi.NewInlineKeyboardButtonURL("Задать вопрос", url),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Восстановить бригаду", "restore"),
		),
	)

	MainTrackFailMessage = fmt.Sprintf(mainTrackFailMessage, link)
	// MainTrackQuizMessage["0_"] = fmt.Sprintf(mainTrackQuizMessage_zero, link)
	MainTrackQuizMessage["1_"] = fmt.Sprintf(mainTrackQuizMessage_one, link)
	FatalSomeThingWrong = fmt.Sprintf(fatalSomeThingWrong, link)
	InfoUnknownCommandMessage = fmt.Sprintf(infoUnknownCommandMessage, link)
	RejectMessage = fmt.Sprintf(rejectMessage, link)
	MainTrackWarnRequiredPhoto = fmt.Sprintf(mainTrackWarnRequiredPhoto, link)
	MainTrackWarnWaitForApprovement = fmt.Sprintf(mainTrackWarnWaitForApprovement, link)
	MainTrackWarnConversationsFinished = fmt.Sprintf(mainTrackWarnConversationsFinished, link)
	RepeatTrackWarnConversationsFinished = fmt.Sprintf(repeatTrackWarnConversationsFinished, link)
	MainTrackAmneziaOvcConfigFormatFileCaption = fmt.Sprintf(mainTrackAmneziaOvcConfigFormatFileCaption, link)

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
	DecisionComments[decisionRejectDoubled] = fmt.Sprintf(decisionCommentsTemplate[decisionRejectDoubled], link)
}
