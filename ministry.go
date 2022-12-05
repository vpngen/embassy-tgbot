package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/crypto/ssh"
)

func GetBrigadier(bot *tgbotapi.BotAPI, chatID int64, ecode string, dept DeptOpts) error {
	cmd := fmt.Sprintf("-ch %s", dept.token)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, dept.controlIP, cmd)

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", dept.controlIP), dept.sshConfig)
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	var b, e bytes.Buffer

	session.Stdout = &b
	session.Stderr = &e

	if err := session.Run(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "session errors:\n%s\n", e.String())

		return fmt.Errorf("ssh run: %w", err)
	}

	r := bufio.NewReader(httputil.NewChunkedReader(&b))

	fullname, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("fullname read: %w", err)
	}

	person, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("person read: %w", err)
	}

	desc64, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("desc64 read: %w", err)
	}

	desc, err := base64.StdEncoding.DecodeString(desc64)
	if err != nil {
		return fmt.Errorf("desc64 decoding: %w", err)
	}

	url64, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("url64 read: %w", err)
	}

	wiki, err := base64.StdEncoding.DecodeString(url64)
	if err != nil {
		return fmt.Errorf("url64 decoding: %w", err)
	}

	mnemo, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("mnemo read: %w", err)
	}

	keydesk, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("keydesk read: %w", err)
	}

	wgconf, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("chunk read: %w", err)
	}

	time.Sleep(3 * time.Second)

	msg := fmt.Sprintf("*%s*\n\nИмя: %s\nПрисуждение премии по физике: _%s_\n%s\n\n",
		strings.Trim(fullname, " \r\n\t"),
		strings.Trim(person, " \r\n\t"),
		strings.Trim(string(desc), " \r\n\t"),
		tgbotapi.EscapeText(tgbotapi.ModeMarkdown, strings.Trim(string(wiki), " \r\n\t")),
	)
	_, err = SendMessage(bot, chatID, 0, msg, ecode)
	if err != nil {
		return fmt.Errorf("send person message: %w", err)
	}

	_, err = SendMessage(bot, chatID, 0, SeedMessage, ecode)
	if err != nil {
		return fmt.Errorf("send seed message: %w", err)
	}

	time.Sleep(1 * time.Second)

	msg = fmt.Sprintf(WordsMessage, strings.Trim(mnemo, " \r\n\t"))
	_, err = SendMessage(bot, chatID, 0, msg, ecode)
	if err != nil {
		return fmt.Errorf("send words message: %w", err)
	}

	time.Sleep(3 * time.Second)

	filename := sanitizeFilename(fullname)

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Name: filename, Bytes: wgconf})
	doc.Caption = fmt.Sprintf("http://[%s]", strings.Trim(keydesk, " \r\n\t"))

	if _, err := bot.Request(doc); err != nil {
		return fmt.Errorf("request doc: %w", err)
	}

	return nil
}
