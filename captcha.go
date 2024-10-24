package main

import (
	"math/rand"
	"strings"
	"time"
)

var (
	CaptchaLivetime = 10 * time.Second

	CaptchaLikesMaxAttempts = 3

	CaptchaEscalationTimes = []time.Duration{
		1 * time.Minute,
		3 * time.Minute,
		5 * time.Minute,
		10 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
	}
)

var typicalReactions = []string{
	"\U0001F44D", "\U0001F44E", "\u2764\uFE0F", "\U0001F525", "\U0001F970", "\U0001F44F", "\U0001F601", "\U0001F631", "\U0001F937\u200D\u2642\uFE0F",
}

var (
	line3 = []string{
		"P.S. Если не успеешь за 10 секунд, у тебя будет еще пара попыток.\n",
		"P.S. У тебя будет еще пара попыток, если не успеешь за 10 секунд.\n",
	}

	line2 = []string{
		"Выбери и поставь реакцию (лайк) с порядковым номером равным ответу:\n\n",
		"Выбери и поставь реакцию (смайлик) с порядковым номером равным ответу:\n\n",
	}

	// "🍓", "🍎", "🍉", "🍊", "🍋", "🍐", "💀", "🌵", "🍄‍🟫", "🥭", "☕️", "🏀", "🚲", "🚜", "🪟",
	line1 = map[string][]string{
		"🍎": {
			"За 10 секунд посчитай, сколько яблок нужно для пирога:\n\n",
			"За 10 секунд посчитай, сколько яблок загадано ниже:\n\n",
		},
		"🍉": {
			"За 10 секунд посчитай, сколько арбузов загадано ниже:\n\n",
		},
		"🍊": {
			"За 10 секунд посчитай, сколько апельсинов загадано ниже:\n\n",
		},
		"🍋": {
			"За 10 секунд посчитай, сколько лимонов загадано ниже:\n\n",
		},
		"🍐": {
			"За 10 секунд посчитай, сколько груш загадано ниже:\n\n",
		},
		"🌵": {
			"За 10 секунд посчитай, сколько кактусов загадано ниже:\n\n",
		},
		"🍄‍🟫": {
			"За 10 секунд посчитай, сколько грибов загадано ниже:\n\n",
		},
		"🥭": {
			"За 10 секунд посчитай, сколько манго загадано ниже:\n\n",
		},
		"🚲": {
			"За 10 секунд посчитай, сколько велосипедов загадано ниже:\n\n",
		},
		"🪟": {
			"За 10 секунд посчитай, какое количество окон в свободный интернет загадано ниже:\n\n",
			"За 10 секунд посчитай, сколько окон загадано ниже:\n\n",
			"За 10 секунд посчитай, сколько окон загадано:\n\n",
			"За 10 секунд посчитай, сколько окон загадано:\n\n",
		},
		"🚜": {
			"За 10 секунд посчитай, сколько тракторов нужно поросёнку Петру, чтобы увезти всю семью:\n\n",
			"За 10 секунд посчитай, сколько тракторов загадано ниже:\n\n",
			"За 10 секунд посчитай, сколько тракторов загадано ниже:\n\n",
			"За 10 секунд посчитай, сколько тракторов загадано:\n\n",
			"За 10 секунд посчитай, сколько тракторов загадано:\n\n",
		},
		"🏀": {
			"За 10 секунд посчитай, сколько мячей нужно забить нашей команде для выигрыша:\n\n",
			"За 10 секунд посчитай, сколько мячей загадано ниже:\n\n",
			"За 10 секунд посчитай, сколько мячей загадано:\n\n",
		},
		"☕️": {
			"За 10 секунд посчитай, сколько чашек кофе выпьет наш программист за день:\n\n",
			"За 10 секунд посчитай, сколько чашек кофе загадано ниже:\n\n",
			"За 10 секунд посчитай, сколько чашек кофе загадано:\n\n",
		},
		"🍓": {
			"За 10 секунд посчитай, сколько клубники нужно для пирога:\n\n",
			"За 10 секунд посчитай, сколько клубники загадано ниже:\n\n",
			"За 10 секунд посчитай, сколько клубники загадано ниже:\n\n",
			"За 10 секунд посчитай, сколько клубники загадано:\n\n",
			"За 10 секунд посчитай, сколько клубники загадано:\n\n",
		},
		"💀": {
			"За 10 секунд посчитай, сколько людей в костюмах черепа пришло на Хэллоуин:\n\n",
			"За 10 секунд посчитай, сколько черепов загадано ниже:\n\n",
			"За 10 секунд посчитай, сколько черепов загадано ниже:\n\n",
			"За 10 секунд посчитай, сколько черепов загадано ниже:\n\n",
			"За 10 секунд посчитай, сколько черепов загадано ниже:\n\n",
			"За 10 секунд посчитай, сколько черепов загадано ниже:\n\n",
		},
	}

	captchaEmoji []string = func() []string {
		res := make([]string, 0, len(line1))

		for k := range line1 {
			res = append(res, k)
		}

		return res
	}()

	captchaOperators = map[string][]string{
		"+": {
			"+",
			"плюс",
			"прибавить",
			"сложить",
			"добавить",
			"➕",
		},
		"-": {
			"-",
			"минус",
			"вычесть",
			"отнять",
			"убавить",
			"➖",
		},
		"=": {
			"равно",
			"получится",
			"результат",
			"будет",
		},
	}
)

const obfsWeight = 5

func captchaObfs(s string) string {
	res := ""

	for _, r := range s {
		switch r {
		case 'а', 'и', 'о', 'к', 'е', 'р', 'с', 'у', 'х', 'ё', 'Н', 'Р', 'А', 'О', 'К', 'Е', 'С', 'У', 'Х', 'Ё':
			if rand.Int31n(obfsWeight) == 0 {
				switch r {
				case 'а':
					r = 'a'
				case 'и':
					r = 'u'
				case 'о':
					r = 'o'
				case 'к':
					r = 'k'
				case 'е':
					r = 'e'
				case 'р':
					r = 'p'
				case 'с':
					r = 'c'
				case 'у':
					r = 'y'
				case 'х':
					r = 'x'
				case 'ё':
					r = 'ë'
				case 'Н':
					r = 'H'
				case 'Р':
					r = 'P'
				case 'А':
					r = 'A'
				case 'О':
					r = 'O'
				case 'К':
					r = 'K'
				case 'Е':
					r = 'E'
				case 'С':
					r = 'C'
				case 'У':
					r = 'Y'
				case 'Х':
					r = 'X'
				case 'Ё':
					r = 'Ë'
				}
			}

			res += string(r)
		default:
			res += string(r)
		}
	}

	return res
}

const (
	captchaFullNumMin = 1
	captchaFullNumMax = 8

	captchaMin = 1 // can't be greater than captchaLikeMax - captchaLikeMin
	captchaMax = 6

	captchaLikeMin = 2
	captchaLikeMax = 4
)

func captchaCreateTerm() (string, int, int, int) {
	x1 := rand.Intn(captchaMax-captchaMin+1) + captchaMin

	if x1 <= captchaLikeMin+captchaMin {
		// important that x1 is not greater than captchaLikeMax
		x2 := rand.Intn(captchaLikeMax-x1) + captchaMin

		if rand.Int31n(2) == 0 {
			return "+", x2, x1, x1 + x2
		}

		return "+", x1, x2, x1 + x2
	}

	x2 := rand.Intn(x1-captchaLikeMin) + captchaMin

	return "-", x1, x2, x1 - x2
}

func captchaFillValue(emoji string, x int) string {
	emoji2 := emoji
	for emoji2 == emoji {
		emoji2 = captchaEmoji[rand.Intn(len(captchaEmoji))]
	}

	delta := rand.Intn(captchaFullNumMax-x) + 1
	l := x + delta

	buf := make([]string, 0, l)

	for i := 0; i < l; i++ {
		buf = append(buf, emoji)
	}

	if delta > 0 {
		pos := make(map[int]struct{}, delta)

		for len(pos) < delta {
			j := rand.Intn(l)

			if _, ok := pos[j]; !ok {
				pos[j] = struct{}{}

				buf[j] = emoji2
			}
		}
	}

	return strings.Join(buf, " ")
}

func captchaTerm(emoji string) (int, string, int, int, string) {
	operator, x1, x2, y := captchaCreateTerm()

	return x1, operator, x2, y,
		captchaFillValue(emoji, x1) + " " +
			captchaObfs(captchaOperators[operator][rand.Intn(len(captchaOperators[operator]))]) + " " +
			captchaFillValue(emoji, x2) + " " +
			captchaObfs(captchaOperators["="][rand.Intn(len(captchaOperators["="]))]) + " " + "?"
}

func captchaFillReactions(x int) (string, string) {
	delta := rand.Intn(captchaMax - x)
	l := x + delta

	if l < 4 {
		l = 3
	}

	buf := make([]string, 0, l)

	pos := make(map[string]struct{}, l)
	for len(pos) < l {
		s := typicalReactions[rand.Intn(len(typicalReactions))]

		if _, ok := pos[s]; !ok {
			pos[s] = struct{}{}

			buf = append(buf, s)
		}
	}

	return buf[x-1], strings.Join(buf, " ")
}

func GetCaptchaText() (string, string) {
	// emoji
	emoji := captchaEmoji[rand.Intn(len(captchaEmoji))]

	_, _, _, y, term := captchaTerm(emoji)

	// line0 := fmt.Sprintf("%d %s %d = %d\n\n", x1, op, x2, y)
	line0 := ""

	// line1
	line1Text := captchaObfs(line1[emoji][rand.Intn(len(line1[emoji]))])

	// line2
	line2Text := captchaObfs(line2[rand.Intn(len(line2))])

	// line3
	line3Text := captchaObfs(line3[rand.Intn(len(line3))])

	// likes
	like, lineLikes := captchaFillReactions(y)

	return like, line1Text + term + "\n\n" + line0 + line2Text + lineLikes + "\n\n" + line3Text
}
