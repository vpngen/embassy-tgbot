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

	"github.com/google/uuid"
	"github.com/vpngen/embassy-tgbot/logs"
	"github.com/vpngen/ministry"
	"golang.org/x/crypto/ssh"
)

// https://t.me/vipgenbot?start=

func reqBrigade(opts MinistryOpts, chatID int64, label SessionLabel) (uuid.UUID, error) {
	logs.Infof("Request brigade from %s\n", opts.controlIP)

	label = setLabel(label, MarkerEmptyLabel)

	telegramID := chatID ^ telegramIDCover

	cmd := fmt.Sprintf("reqvipid -ch -tgid %d -l %s -lt %d -lu %s %s", telegramID, label.Label, label.Time.Unix(), label.ID.String(), opts.token)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, opts.controlIP, cmd)

	if opts.fake {
		logs.Debugf("fake msg tread\n")

		return uuid.Nil, nil
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
		return uuid.Nil, fmt.Errorf("json unmarshal: %w", err)
	}

	return req.RequestID, nil
}
