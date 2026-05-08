package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/vpngen/embassy-tgbot/logs"
	"github.com/vpngen/ministry"
	"golang.org/x/crypto/ssh"
)

// https://t.me/vipgenbot?start=

type ministryReserveRequest struct {
	BrigadeID    uuid.UUID `json:"brigade_id"`
	UserIdentity string    `json:"user_identity"`
}

type ministryReserveResponse struct {
	OK bool `json:"ok"`
}

// reserveVIPWithMinistry links the brigade UUID to the Telegram user via the ministry-api /reserve endpoint.
func reserveVIPWithMinistry(apiURL, token string, brigadeUUID uuid.UUID, chatID int64) error {
	if brigadeUUID == uuid.Nil {
		return nil // fake/test mode
	}

	body, err := json.Marshal(ministryReserveRequest{
		BrigadeID:    brigadeUUID,
		UserIdentity: strconv.FormatInt(chatID, 10),
	})
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL+"/reserve", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ministry api status %d", resp.StatusCode)
	}

	var res ministryReserveResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !res.OK {
		return fmt.Errorf("ministry api: reserve not ok")
	}

	return nil
}

func reqBrigade(opts MinistryOpts, chatID int64, label SessionLabel, bid string) (uuid.UUID, error) {
	logs.Infof("Request brigade from %s\n", opts.controlIP)

	label = setLabel(label, MarkerEmptyLabel)

	telegramID := chatID ^ telegramIDCover

	if bid != "" {
		bid = fmt.Sprintf("-bid %s", bid)
	}

	cmd := fmt.Sprintf("reqvipid -ch -tgid %d -l %s -lt %d -lu %s %s %s", telegramID, label.Label, label.Time.Unix(), label.ID.String(), bid, opts.token)

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
