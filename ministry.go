package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http/httputil"
	"net/netip"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vpngen/embassy-tgbot/internal/kdlib"
	"github.com/vpngen/wordsgens/namesgenerator"
	"github.com/vpngen/wordsgens/seedgenerator"
	"golang.org/x/crypto/ssh"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	qrcode "github.com/skip2/go-qrcode"
)

const (
	fakeSeedPrefix    = "телеграмживи"
	fakeKeydeskPrefix = "fc00::beaf:0/112"
	fakeEndpointNet   = "182.31.10.0/24"
	fakeCGNAT         = "100.64.0.0/10"
	fakeULA           = "fd00::/8"
)

type grantPkg struct {
	fullname string
	person   string
	desc     string
	wiki     string
	mnemo    string
	keydesk  string
	filename string
	wgconf   []byte
}

var ErrBrigadeNotFound = errors.New("brigade not found")

// SendBrigadierGrants - send grants messages.
func SendBrigadierGrants(bot *tgbotapi.BotAPI, chatID int64, ecode string, opts *grantPkg) error {
	msg := fmt.Sprintf(MainTrackGrantMessage, opts.fullname)
	_, err := SendOpenMessage(bot, chatID, 0, msg, ecode)
	if err != nil {
		return fmt.Errorf("send grant message: %w", err)
	}

	time.Sleep(2 * time.Second)

	msg = fmt.Sprintf(MainTrackPersonDescriptionMessage,
		strings.Trim(opts.person, " \r\n\t"),
		strings.Trim(string(opts.desc), " \r\n\t"),
		tgbotapi.EscapeText(tgbotapi.ModeMarkdown, strings.Trim(string(opts.wiki), " \r\n\t")),
	)
	_, err = SendOpenMessage(bot, chatID, 0, msg, ecode)
	if err != nil {
		return fmt.Errorf("send person message: %w", err)
	}

	time.Sleep(2 * time.Second)

	_, err = SendOpenMessage(bot, chatID, 0, MainTrackSeedDescMessage, ecode)
	if err != nil {
		return fmt.Errorf("send seed message: %w", err)
	}

	time.Sleep(2 * time.Second)

	msg = fmt.Sprintf(MainTrackWordsMessage, strings.Trim(opts.mnemo, " \r\n\t"))
	_, err = SendOpenMessage(bot, chatID, 0, msg, ecode)
	if err != nil {
		return fmt.Errorf("send words message: %w", err)
	}

	time.Sleep(3 * time.Second)

	msg = fmt.Sprintf(MainTrackConfigFormatTextTemplate, string(opts.wgconf))
	_, err = SendOpenMessage(bot, chatID, 0, msg, ecode)
	if err != nil {
		return fmt.Errorf("send text config: %w", err)
	}

	time.Sleep(2 * time.Second)

	png, err := qrcode.Encode(string(opts.wgconf), qrcode.Low, 256)
	if err != nil {
		return fmt.Errorf("create qr: %w", err)
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: opts.filename, Bytes: png})
	photo.Caption = MainTrackConfigFormatQRCaption
	photo.ParseMode = tgbotapi.ModeMarkdown

	if _, err := bot.Request(photo); err != nil {
		return fmt.Errorf("send qr config: %w", err)
	}

	time.Sleep(2 * time.Second)

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Name: opts.filename, Bytes: opts.wgconf})
	doc.Caption = MainTrackConfigFormatFileCaption
	doc.ParseMode = tgbotapi.ModeMarkdown

	if _, err := bot.Request(doc); err != nil {
		return fmt.Errorf("send file config: %w", err)
	}

	time.Sleep(3 * time.Second)

	_, err = SendOpenMessage(bot, chatID, 0, fmt.Sprintf(MainTrackConfigsMessage, opts.keydesk), ecode)
	if err != nil {
		return fmt.Errorf("send keydesk message: %w", err)
	}

	//	time.Sleep(2 * time.Second)

	//	_, err = SendOpenMessage(bot, chatID, 0, fmt.Sprintf(MainTrackKeydeskIPv6Message, opts.keydesk), ecode)
	//	if err != nil {
	//		return fmt.Errorf("send seed message: %w", err)
	//	}

	return nil
}

// SendRestoredBrigadierGrants - send grants messages.
func SendRestoredBrigadierGrants(bot *tgbotapi.BotAPI, chatID int64, ecode string, opts *grantPkg) error {
	_, err := SendOpenMessage(bot, chatID, 0, RestoreTrackGrantMessage, ecode)
	if err != nil {
		return fmt.Errorf("send restore grant message: %w", err)
	}

	time.Sleep(2 * time.Second)

	msg := fmt.Sprintf(MainTrackConfigFormatTextTemplate, string(opts.wgconf))
	_, err = SendOpenMessage(bot, chatID, 0, msg, ecode)
	if err != nil {
		return fmt.Errorf("send text config: %w", err)
	}

	time.Sleep(2 * time.Second)

	png, err := qrcode.Encode(string(opts.wgconf), qrcode.Low, 256)
	if err != nil {
		return fmt.Errorf("create qr: %w", err)
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: opts.filename, Bytes: png})
	photo.Caption = MainTrackConfigFormatQRCaption
	photo.ParseMode = tgbotapi.ModeMarkdown

	if _, err := bot.Request(photo); err != nil {
		return fmt.Errorf("send qr config: %w", err)
	}

	time.Sleep(2 * time.Second)

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Name: opts.filename, Bytes: opts.wgconf})
	doc.Caption = MainTrackConfigFormatFileCaption
	doc.ParseMode = tgbotapi.ModeMarkdown

	if _, err := bot.Request(doc); err != nil {
		return fmt.Errorf("send file config: %w", err)
	}

	time.Sleep(3 * time.Second)

	_, err = SendOpenMessage(bot, chatID, 0, fmt.Sprintf(MainTrackConfigsMessage, opts.keydesk), ecode)
	if err != nil {
		return fmt.Errorf("send keydesk message: %w", err)
	}

	time.Sleep(2 * time.Second)

	domain := "[выдаваемый домен]"
	lines := strings.Split(string(opts.wgconf), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Endpoint") {
			_, d, _ := strings.Cut(line, "=")
			d = strings.Trim(d, " \r\n\t")
			d, _, _ = strings.Cut(d, ":")
			if d != "" {
				domain = d
			}
		}
	}

	if _, err := netip.ParseAddr(domain); err == nil {
		hint := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Bytes: RestoreTrackImgVgbs})
		hint.Caption = fmt.Sprintf(RestoreTracIP2DomainHintsMessage, domain)
		hint.ParseMode = tgbotapi.ModeMarkdown

		if _, err := bot.Request(hint); err != nil {
			return fmt.Errorf("send hint: %w", err)
		}
	}

	return nil
}

func callMinistry(dept DeptOpts) (*grantPkg, error) {
	opts := &grantPkg{}

	cmd := fmt.Sprintf("createbrigade -ch %s", dept.token)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, dept.controlIP, cmd)

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", dept.controlIP), dept.sshConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	var b, e bytes.Buffer

	session.Stdout = &b
	session.Stderr = &e

	LogTag := "tgembass"
	defer func() {
		switch errstr := e.String(); errstr {
		case "":
			fmt.Fprintf(os.Stderr, "%s: SSH Session StdErr: empty\n", LogTag)
		default:
			fmt.Fprintf(os.Stderr, "%s: SSH Session StdErr:\n", LogTag)
			for _, line := range strings.Split(errstr, "\n") {
				fmt.Fprintf(os.Stderr, "%s: | %s\n", LogTag, line)
			}
		}
	}()

	if err := session.Run(cmd); err != nil {
		return nil, fmt.Errorf("ssh run: %w", err)
	}

	r := bufio.NewReader(httputil.NewChunkedReader(&b))

	fullname, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("fullname read: %w", err)
	}

	opts.fullname = strings.Trim(fullname, "\r\n\t ")

	person, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("person read: %w", err)
	}

	opts.person = strings.Trim(person, "\r\n\t ")

	desc64, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("desc64 read: %w", err)
	}

	desc, err := base64.StdEncoding.DecodeString(desc64)
	if err != nil {
		return nil, fmt.Errorf("desc64 decoding: %w", err)
	}

	opts.desc = string(desc)

	url64, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("url64 read: %w", err)
	}

	wiki, err := base64.StdEncoding.DecodeString(url64)
	if err != nil {
		return nil, fmt.Errorf("url64 decoding: %w", err)
	}

	opts.wiki = string(wiki)

	mnemo, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("mnemo read: %w", err)
	}

	opts.mnemo = strings.Trim(mnemo, "\r\n\t ")

	keydesk, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("keydesk read: %w", err)
	}

	opts.keydesk = strings.Trim(keydesk, "\r\n\t ")

	filename, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("filename read: %w", err)
	}

	opts.filename = strings.Trim(filename, "\r\n\t ")

	opts.wgconf, err = io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("chunk read: %w", err)
	}

	return opts, nil
}

func callMinistryRestore(dept DeptOpts, name, words string) (*grantPkg, error) {
	opts := &grantPkg{}

	base64name := base64.StdEncoding.EncodeToString([]byte(name))
	base64words := base64.StdEncoding.EncodeToString([]byte(words))

	cmd := fmt.Sprintf("restorebrigadier -ch %s %s", base64name, base64words)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, dept.controlIP, cmd)

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", dept.controlIP), dept.sshConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	var b, e bytes.Buffer

	session.Stdout = &b
	session.Stderr = &e

	LogTag := "tgembass"
	defer func() {
		switch errstr := e.String(); errstr {
		case "":
			fmt.Fprintf(os.Stderr, "%s: SSH Session StdErr: empty\n", LogTag)
		default:
			fmt.Fprintf(os.Stderr, "%s: SSH Session StdErr:\n", LogTag)
			for _, line := range strings.Split(errstr, "\n") {
				fmt.Fprintf(os.Stderr, "%s: | %s\n", LogTag, line)
			}
		}
	}()

	if err := session.Run(cmd); err != nil {
		return nil, fmt.Errorf("ssh run: %w", err)
	}

	r := bufio.NewReader(httputil.NewChunkedReader(&b))

	status, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("status read: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s: Returned status: %s\n", LogTag, status)

	if strings.Trim(status, "\r\n\t ") != "WGCONFIG" {
		return nil, fmt.Errorf("status: %s: %w", status, ErrBrigadeNotFound)
	}

	keydesk, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("keydesk read: %w", err)
	}

	opts.keydesk = strings.Trim(keydesk, "\r\n\t ")

	filename, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("filename read: %w", err)
	}

	opts.filename = strings.Trim(filename, "\r\n\t ")

	opts.wgconf, err = io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("wgconf read: %w", err)
	}

	return opts, nil
}

// GetBrigadier - get brigadier name and config.
func GetBrigadier(bot *tgbotapi.BotAPI, chatID int64, ecode string, dept DeptOpts) error {
	var (
		opts *grantPkg
		err  error
	)

	switch dept.fake {
	case false:
		opts, err = callMinistry(dept)
		if err != nil {
			return fmt.Errorf("call ministry: %w", err)
		}
	case true:
		opts, err = genGrants(dept)
		if err != nil {
			return fmt.Errorf("gen grants: %w", err)
		}
	}

	time.Sleep(3 * time.Second)

	err = SendBrigadierGrants(bot, chatID, ecode, opts)
	if err != nil {
		return fmt.Errorf("send grants: %w", err)
	}

	return nil
}

func MyTitle(s string) string {
	// Use a closure here to remember state.
	// Hackish but effective. Depends on Map scanning in order and calling
	// the closure once per rune.
	prev := ' '
	return strings.Map(
		func(r rune) rune {
			if r != ' ' && prev == ' ' || prev == '-' || prev == '_' || prev == '.' {
				prev = r
				return unicode.ToTitle(r)
			}
			prev = r
			return r
		},
		s)
}

const maxEYoCombinations = 9

func generateCombinations(s string, max int) []string {
	return replaceEWithYo(s, 0, max)
}

func replaceEWithYo(s string, start, max int) []string {
	if start >= len(s) || max <= 0 {
		return []string{s}
	}

	r, size := utf8.DecodeRuneInString(s[start:])
	if r == 'е' || r == 'ё' {
		eStr := replaceRuneAt(s, start, size, "е")
		yoStr := replaceRuneAt(s, start, size, "ё")

		return append(
			replaceEWithYo(eStr, start+size, (max-1)/2),
			replaceEWithYo(yoStr, start+size, (max-1)/2)...,
		)
	} else {
		return replaceEWithYo(s, start+size, max)
	}
}

func replaceRuneAt(s string, index, size int, replacement string) string {
	return s[:index] + replacement + s[index+size:]
}

// RestoreBrigadier - restore brigadier  config.
func RestoreBrigadier(bot *tgbotapi.BotAPI, chatID int64, ecode string, dept DeptOpts, name, words string) error {
	var (
		opts *grantPkg
		err  error
	)

	switch dept.fake {
	case false:
		opts, err = callMinistryRestore(dept, name, words)
		if err == nil {
			break
		}

		words = strings.Replace(strings.ToLower(words), "ё", "е", -1)

		fmt.Fprintf(os.Stderr, "Try name/words: %s %s\n", name, words)

		opts, err = callMinistryRestore(dept, name, words)
		if err == nil {
			break
		}

		name = MyTitle(strings.ToLower(name))

		fmt.Fprintf(os.Stderr, "Try name/words: %s %s\n", name, words)

		opts, err = callMinistryRestore(dept, name, words)
		if err == nil {
			break
		}

		for _, name := range generateCombinations(name, maxEYoCombinations) {
			fmt.Fprintf(os.Stderr, "Try name/words: %s %s\n", name, words)

			opts, err = callMinistryRestore(dept, name, words)
			if err == nil {
				break
			}
		}

		if err != nil {
			return fmt.Errorf("call ministry: %w", err)
		}

	case true:
		opts, err = genGrants(dept)
		if err != nil {
			return fmt.Errorf("gen grants: %w", err)
		}
	}

	time.Sleep(3 * time.Second)

	err = SendRestoredBrigadierGrants(bot, chatID, ecode, opts)
	if err != nil {
		return fmt.Errorf("send grants: %w", err)
	}

	return nil
}

func genGrants(dept DeptOpts) (*grantPkg, error) {
	opts := &grantPkg{}

	fullname, person, err := namesgenerator.PhysicsAwardeeShort()
	if err != nil {
		return nil, fmt.Errorf("physics gen: %w", err)
	}

	opts.fullname = fullname
	opts.person = person.Name
	opts.desc = person.Desc
	opts.wiki = person.URL

	opts.mnemo, _, _, err = seedgenerator.Seed(seedgenerator.ENT64, fakeSeedPrefix)
	if err != nil {
		return nil, fmt.Errorf("gen seed6: %w", err)
	}

	opts.keydesk = kdlib.RandomAddrIPv6(netip.MustParsePrefix(fakeKeydeskPrefix)).String()

	numbered := fmt.Sprintf("%03d %s", rand.Int31n(256), fullname)
	opts.filename = kdlib.SanitizeFilename(numbered)

	wgkey, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("gen wg psk: %w", err)
	}

	wgpriv, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("gen wg psk: %w", err)
	}

	wgpub := wgpriv.PublicKey()

	tmpl := `[Interface]
Address = %s
PrivateKey = %s
DNS = %s

[Peer]
Endpoint = %s:51820
PublicKey = %s
PresharedKey = %s
AllowedIPs = 0.0.0.0/0,::/0
`

	ipv4 := kdlib.RandomAddrIPv4(netip.MustParsePrefix(fakeCGNAT))
	ipv6 := kdlib.RandomAddrIPv6(netip.MustParsePrefix(fakeULA))
	ep := kdlib.RandomAddrIPv4(netip.MustParsePrefix(fakeEndpointNet))

	opts.wgconf = fmt.Appendf(opts.wgconf,
		tmpl,
		netip.PrefixFrom(ipv4, 32).String()+","+netip.PrefixFrom(ipv6, 128).String(),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgpriv[:]),
		ipv4.String()+","+ipv6.String(),
		ep.String(),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgpub[:]),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgkey[:]),
	)

	return opts, nil
}
