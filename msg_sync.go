package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/embassy-tgbot/logs"
	tgbotapi "github.com/vpngen/embassy-tgbot/telegram-bot-api"
	"github.com/vpngen/ministry"
	"golang.org/x/crypto/ssh"
)

const (
	MsgSyncDuration       = time.Minute
	MsgReadDuration       = time.Second
	telegramIDCover int64 = 24537551337805
)

func msgSyncLoop(wg *sync.WaitGroup, bot *tgbotapi.BotAPI, stop <-chan struct{}, opts MinistryOpts) {
	defer wg.Done()

	tm := time.NewTimer(time.Second)

	defer tm.Stop()

	for {
		select {
		case <-tm.C:
			logs.Info("Start msg sync\n")

			msg, err := readMsg(opts)
			if err != nil {
				logs.Errf("logs send: %s\n", err)
			}

			if msg == nil {
				tm.Reset(MsgSyncDuration)

				continue
			}

			chatID := msg.TelegramID ^ telegramIDCover

			wg := &sync.WaitGroup{}

			ecode := genEcode()

			logs.Warningf("New msg to %s (%d)\n", msg.Name, chatID)

			wg.Add(1)
			if err := SendBrigadierGrants(bot, wg, MainTrackGrantMessageVIP, chatID, ecode, &msg.Answer); err != nil {
				logs.Errf("send grants: %s", err)

				tm.Reset(MsgReadDuration)

				continue
			}
			wg.Wait()

			if _, err = SendOpenMessage(bot, chatID, 0, false, MainTrackGrantSupportMessageVIP, ecode); err != nil {
				logs.Errf("logs send: %s\n", err)
			}

			if err := doneMsg(opts, msg.RequestID); err != nil {
				logs.Errf("done msg: %s", err)

				tm.Reset(MsgReadDuration)

				continue
			}

			tm.Reset(MsgReadDuration)
		case <-stop:
			return
		}
	}
}

func readMsg(opts MinistryOpts) (*ministry.VIPAnswer, error) {
	logs.Infof("readMsg\n")

	cmd := fmt.Sprintf("readmsgs -ch %s", opts.token)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, opts.controlIP, cmd)

	if opts.fake {
		logs.Debugf("fake msg tread\n")

		return nil, nil
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", opts.controlIP), opts.sshConfig)
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
		return nil, fmt.Errorf("start: %w", err)
	}

	r := bufio.NewReader(httputil.NewChunkedReader(&b))

	payload, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("chunk read: %w", err)
	}

	wgconf := &ministry.VIPAnswer{}
	if err := json.Unmarshal(payload, &wgconf); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	return wgconf, nil
}

func doneMsg(opts MinistryOpts, bid uuid.UUID) error {
	logs.Infof("doneMsg\n")

	cmd := fmt.Sprintf("readmsgs -id %s -ch %s", bid, opts.token)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, opts.controlIP, cmd)

	if opts.fake {
		logs.Debugf("fake msg tread\n")

		return nil
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", opts.controlIP), opts.sshConfig)
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
		return fmt.Errorf("start: %w", err)
	}

	return nil
}
