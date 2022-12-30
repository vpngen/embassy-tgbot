package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http/httputil"
	"net/netip"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vpngen/keydesk/user"
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

func SendBrigadierGrants(bot *tgbotapi.BotAPI, chatID int64, ecode string, opts *grantPkg) error {
	msg := fmt.Sprintf(GrantMessage, opts.fullname)
	_, err := SendOpenMessage(bot, chatID, 0, msg, ecode)
	if err != nil {
		return fmt.Errorf("send grant message: %w", err)
	}

	time.Sleep(2 * time.Second)

	msg = fmt.Sprintf("*Справка*\n\nЛауреат нобелевской премии по физике: *%s*\n_%s_\n%s\n\n",
		strings.Trim(opts.person, " \r\n\t"),
		strings.Trim(string(opts.desc), " \r\n\t"),
		tgbotapi.EscapeText(tgbotapi.ModeMarkdown, strings.Trim(string(opts.wiki), " \r\n\t")),
	)
	_, err = SendOpenMessage(bot, chatID, 0, msg, ecode)
	if err != nil {
		return fmt.Errorf("send person message: %w", err)
	}

	time.Sleep(2 * time.Second)

	_, err = SendOpenMessage(bot, chatID, 0, SeedMessage, ecode)
	if err != nil {
		return fmt.Errorf("send seed message: %w", err)
	}

	time.Sleep(2 * time.Second)

	msg = fmt.Sprintf(WordsMessage, strings.Trim(opts.mnemo, " \r\n\t"))
	_, err = SendOpenMessage(bot, chatID, 0, msg, ecode)
	if err != nil {
		return fmt.Errorf("send words message: %w", err)
	}

	time.Sleep(3 * time.Second)

	_, err = SendOpenMessage(bot, chatID, 0, KeydeskMessage, ecode)
	if err != nil {
		return fmt.Errorf("send seed message: %w", err)
	}

	time.Sleep(2 * time.Second)

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Name: opts.filename, Bytes: opts.wgconf})
	doc.Caption = "Конфигурация файлом"

	if _, err := bot.Request(doc); err != nil {
		return fmt.Errorf("request doc: %w", err)
	}

	time.Sleep(2 * time.Second)

	msg = fmt.Sprintf("Конфигурация текстом:\n```%s```", "\n"+string(opts.wgconf))
	_, err = SendOpenMessage(bot, chatID, 0, msg, ecode)
	if err != nil {
		return fmt.Errorf("send seed message: %w", err)
	}

	time.Sleep(2 * time.Second)

	png, err := qrcode.Encode(string(opts.wgconf), qrcode.Low, 256)
	if err != nil {
		return fmt.Errorf("create qr: %w", err)
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: opts.filename, Bytes: png})
	photo.Caption = "Конфигурация QR-кодом"

	if _, err := bot.Request(photo); err != nil {
		return fmt.Errorf("request photo: %w", err)
	}

	return nil
}

func callMinistry(dept DeptOpts) (*grantPkg, error) {
	var opts = &grantPkg{}

	cmd := fmt.Sprintf("-ch %s", dept.token)

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

	if err := session.Run(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "session errors:\n%s\n", e.String())

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

func genGrants(dept DeptOpts) (*grantPkg, error) {
	var opts = &grantPkg{}

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

	opts.keydesk = user.RandomAddrIPv6(netip.MustParsePrefix(fakeKeydeskPrefix)).String()

	numbered := fmt.Sprintf("%03d %s", rand.Int31n(256), fullname)
	opts.filename = user.SanitizeFilename(numbered)

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

	ipv4 := user.RandomAddrIPv4(netip.MustParsePrefix(fakeCGNAT))
	ipv6 := user.RandomAddrIPv6(netip.MustParsePrefix(fakeULA))
	ep := user.RandomAddrIPv4(netip.MustParsePrefix(fakeEndpointNet))

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
