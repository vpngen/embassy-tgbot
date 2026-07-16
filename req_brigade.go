package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/vpngen/embassy-tgbot/logs"
	"github.com/vpngen/ministry"
	"golang.org/x/crypto/ssh"
)

type reserveVIPResponse struct {
	OK bool `json:"ok"`
}

// reserveVIPWithMinistry links the brigade UUID to the Telegram user via the
// ministry reservebrigade SSH command.
func reserveVIPWithMinistry(opts MinistryOpts, brigadeUUID uuid.UUID, chatID int64) error {
	if brigadeUUID == uuid.Nil {
		return nil
	}

	userIdentity := strconv.FormatInt(chatID, 10)
	cmd := fmt.Sprintf("reservebrigade -ch -j %s %s %s", opts.token, brigadeUUID.String(), userIdentity)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, opts.controlIP, cmd)

	if opts.fake {
		logs.Debugf("fake reserve vip\n")
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

	const logTag = "tgembass-reservevip"
	defer func() {
		switch errstr := e.String(); errstr {
		case "":
			fmt.Fprintf(os.Stderr, "%s: SSH Session StdErr: empty\n", logTag)
		default:
			fmt.Fprintf(os.Stderr, "%s: SSH Session StdErr:\n", logTag)
			for _, line := range strings.Split(errstr, "\n") {
				fmt.Fprintf(os.Stderr, "%s: | %s\n", logTag, line)
			}
		}
	}()

	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	r := bufio.NewReader(httputil.NewChunkedReader(&b))

	payload, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("chunk read: %w", err)
	}

	var res reserveVIPResponse
	if err := json.Unmarshal(payload, &res); err != nil {
		return fmt.Errorf("reserve vip json unmarshal: %w", err)
	}

	if !res.OK {
		return fmt.Errorf("reserve: not ok")
	}

	return nil
}

func reqBrigade(opts MinistryOpts, chatID int64, label SessionLabel, bid, lang string) (uuid.UUID, error) {
	logs.Infof("Request brigade from %s\n", opts.controlIP)

	label = setLabel(label, MarkerEmptyLabel)

	telegramID := chatID ^ telegramIDCover

	if bid != "" {
		bid = fmt.Sprintf("-bid %s", bid)
	}

	if lang == "" {
		lang = "ru"
	}

	cmd := fmt.Sprintf("reqvipid -ch -tgid %d -l %s -lt %d -lu %s -lang %s %s %s", telegramID, label.Label, label.Time.Unix(), label.ID.String(), lang, bid, opts.token)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, opts.controlIP, cmd)

	if opts.fake {
		logs.Debugf("fake msg tread\n")

		return uuid.New(), nil
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", opts.controlIP), opts.sshConfig)
	if err != nil {
		return uuid.Nil, fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return uuid.Nil, fmt.Errorf("ssh session: %w", err)
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
		return uuid.Nil, fmt.Errorf("start: %w", err)
	}

	r := bufio.NewReader(httputil.NewChunkedReader(&b))

	payload, err := io.ReadAll(r)
	if err != nil {
		return uuid.Nil, fmt.Errorf("chunk read: %w", err)
	}

	req := ministry.VIPReserve{}
	if err := json.Unmarshal(payload, &req); err != nil {
		return uuid.Nil, fmt.Errorf("req brigade json unmarshal: %w", err)
	}

	return req.RequestID, nil
}
