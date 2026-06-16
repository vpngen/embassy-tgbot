package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
	"github.com/vpngen/embassy-tgbot/logs"
	tgbotapi "github.com/vpngen/embassy-tgbot/telegram-bot-api"
	"golang.org/x/crypto/ssh"
)

const (
	PushSyncDuration = time.Minute
	PushReadDuration = time.Second
)

const blockedPrefix = "tgblocked_"

// PushAnswer mirrors the response from the readpush SSH command.
type PushAnswer struct {
	TelegramID int64     `json:"telegram_id"`
	RequestID  uuid.UUID `json:"request_id"`
	EventType  string    `json:"event_type"`
	Lang       string    `json:"lang"`
}

const (
	pushKeyNotActivated   = "push_not_activated"
	pushKeyLowUsers1d     = "push_low_users_1d"
	pushKeyLowUsers3d     = "push_low_users_3d"
	pushKeyLastChance     = "push_last_chance"
	pushKeyBrigadeDeleted = "push_brigade_deleted"
)

const (
	pushKeyVIPBuyKey        = "push_vip_buy_key"
	pushKeyVIPRenewalFailed = "push_vip_renewal_failed"
	pushKeyVIPLastChance    = "push_vip_last_chance"
	pushKeyVIPExpired       = "push_vip_expired"
)

func blockedKey(chatID int64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(chatID))

	return append([]byte(blockedPrefix), b[:]...)
}

func isBlocked(db *badger.DB, chatID int64) bool {
	err := db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(blockedKey(chatID))

		return err
	})

	return err == nil
}

func setBlocked(db *badger.DB, chatID int64) error {
	return db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(blockedKey(chatID), []byte{1})

		return txn.SetEntry(e)
	})
}

func clearBlocked(db *badger.DB, chatID int64) {
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Delete(blockedKey(chatID))
	})

	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		logs.Errf("clearBlocked %d: %s\n", chatID, err)
	}
}

func pushSyncLoop(wg *sync.WaitGroup, bot *tgbotapi.BotAPI, stop <-chan struct{}, db *badger.DB, opts MinistryOpts, flowMainUrl string) {
	defer wg.Done()

	fmt.Fprintf(os.Stderr, "pushSyncLoop: start\n")

	tm := time.NewTimer(time.Second)
	defer tm.Stop()

	for {
		select {
		case <-tm.C:
			logs.Info("Start push sync\n")

			push, err := readPush(opts)
			if err != nil {
				logs.Errf("push read: %s\n", err)
			}

			if push == nil {
				tm.Reset(PushSyncDuration)

				continue
			}

			chatID := push.TelegramID ^ telegramIDCover

			logs.Warningf("New push to %d (event: %s)\n", chatID, push.EventType)

			if isBlocked(db, chatID) {
				logs.Warningf("push skip: chat %d has blocked the bot\n", chatID)

				if err := donePush(opts, push.RequestID, push.EventType); err != nil {
					logs.Errf("push done (blocked): %s\n", err)
				}

				tm.Reset(PushReadDuration)

				continue
			}

			msg := ministryMessage(flowMainUrl, pushFlowKey(push.EventType), "", push.Lang)
			if msg == "" {
				logs.Errf("push send: empty message for event %s\n", push.EventType)

				tm.Reset(PushSyncDuration)

				continue
			}

			ecode := genEcode()

			if _, err := SendOpenMessage(bot, chatID, 0, false, msg, ecode); err != nil {
				if IsForbiddenError(err) {
					logs.Warningf("push send: chat %d blocked bot (403), recording\n", chatID)

					if dbErr := setBlocked(db, chatID); dbErr != nil {
						logs.Errf("setBlocked %d: %s\n", chatID, dbErr)
					}

					if err := donePush(opts, push.RequestID, push.EventType); err != nil {
						logs.Errf("push done (403): %s\n", err)
					}

					tm.Reset(PushReadDuration)

					continue
				}

				if ok, wait := IsRateLimitedError(err); ok {
					logs.Warningf("push send: rate limited by telegram, retrying after %s\n", wait)

					tm.Reset(wait)

					continue
				}

				logs.Errf("push send: %s\n", err)

				tm.Reset(PushReadDuration)

				continue
			}

			if err := donePush(opts, push.RequestID, push.EventType); err != nil {
				logs.Errf("push done: %s\n", err)

				tm.Reset(PushReadDuration)

				continue
			}

			tm.Reset(PushReadDuration)
		case <-stop:
			return
		}
	}
}

func pushFlowKey(eventType string) string {
	switch eventType {
	case "free.not_activated":
		return pushKeyNotActivated
	case "free.low_users_1d":
		return pushKeyLowUsers1d
	case "free.low_users_3d":
		return pushKeyLowUsers3d
	case "free.last_chance":
		return pushKeyLastChance
	case "free.brigade_deleted":
		return pushKeyBrigadeDeleted
	case "vip.buy_vip_key":
		return pushKeyVIPBuyKey
	case "vip.renewal_failed":
		return pushKeyVIPRenewalFailed
	case "vip.last_chance":
		return pushKeyVIPLastChance
	case "vip.subscription_expired":
		return pushKeyVIPExpired
	default:
		return ""
	}
}

func pushVipSyncLoop(wg *sync.WaitGroup, bot *tgbotapi.BotAPI, stop <-chan struct{}, db *badger.DB, opts MinistryOpts, flowMainUrl string) {
	defer wg.Done()

	fmt.Fprintf(os.Stderr, "pushVipSyncLoop: start\n")

	tm := time.NewTimer(time.Second)
	defer tm.Stop()

	for {
		select {
		case <-tm.C:
			logs.Info("Start vip push sync\n")

			push, err := readPushVIP(opts)
			if err != nil {
				logs.Errf("vip push read: %s\n", err)
			}

			if push == nil {
				tm.Reset(PushSyncDuration)

				continue
			}

			chatID := push.TelegramID ^ telegramIDCover

			logs.Warningf("New VIP push to %d (event: %s)\n", chatID, push.EventType)

			if isBlocked(db, chatID) {
				logs.Warningf("vip push skip: chat %d has blocked the bot\n", chatID)

				if err := donePushVIP(opts, push.RequestID, push.EventType); err != nil {
					logs.Errf("vip push done (blocked): %s\n", err)
				}

				tm.Reset(PushReadDuration)

				continue
			}

			msg := ministryMessage(flowMainUrl, pushFlowKey(push.EventType), "", push.Lang)
			if msg == "" {
				logs.Errf("vip push send: empty message for event %s\n", push.EventType)

				tm.Reset(PushSyncDuration)

				continue
			}

			logs.Infof("Send push message: %s to %d", msg, chatID)

			// ecode := genEcode()

			// if _, err := SendOpenMessage(bot, chatID, 0, false, msg, ecode); err != nil {
			// 	if IsForbiddenError(err) {
			// 		logs.Warningf("vip push send: chat %d blocked bot (403), recording\n", chatID)

			// 		if dbErr := setBlocked(db, chatID); dbErr != nil {
			// 			logs.Errf("setBlocked %d: %s\n", chatID, dbErr)
			// 		}

			// 		if err := donePushVIP(opts, push.RequestID, push.EventType); err != nil {
			// 			logs.Errf("vip push done (403): %s\n", err)
			// 		}

			// 		tm.Reset(PushReadDuration)

			// 		continue
			// 	}

			// 	if ok, wait := IsRateLimitedError(err); ok {
			// 		logs.Warningf("vip push send: rate limited by telegram, retrying after %s\n", wait)

			// 		tm.Reset(wait)

			// 		continue
			// 	}

			// 	logs.Errf("vip push send: %s\n", err)

			// 	tm.Reset(PushReadDuration)

			// 	continue
			// }

			if err := donePushVIP(opts, push.RequestID, push.EventType); err != nil {
				logs.Errf("vip push done: %s\n", err)

				tm.Reset(PushReadDuration)

				continue
			}

			tm.Reset(PushReadDuration)
		case <-stop:
			return
		}
	}
}

func readPushVIP(opts MinistryOpts) (*PushAnswer, error) {
	logs.Infof("readPushVIP\n")

	cmd := fmt.Sprintf("readpushvip -ch %s", opts.token)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, opts.controlIP, cmd)

	if opts.fake {
		logs.Debugf("fake vip push read\n")

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
		return nil, fmt.Errorf("ssh run: %w", err)
	}

	r := bufio.NewReader(httputil.NewChunkedReader(&b))

	payload, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("chunk read: %w", err)
	}

	var push *PushAnswer
	if err := json.Unmarshal(payload, &push); err != nil {
		return nil, fmt.Errorf("vip push payload json unmarshal: %w", err)
	}

	return push, nil
}

func donePushVIP(opts MinistryOpts, requestID uuid.UUID, eventType string) error {
	logs.Infof("donePushVIP\n")

	cmd := fmt.Sprintf("readpushvip -id %s -event %s %s", requestID, eventType, opts.token)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, opts.controlIP, cmd)

	if opts.fake {
		logs.Debugf("fake vip push done\n")

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
		return fmt.Errorf("ssh run: %w", err)
	}

	return nil
}

func readPush(opts MinistryOpts) (*PushAnswer, error) {
	logs.Infof("readPush\n")

	cmd := fmt.Sprintf("readpush -ch %s", opts.token)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, opts.controlIP, cmd)

	if opts.fake {
		logs.Debugf("fake push read\n")

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
		return nil, fmt.Errorf("ssh run: %w", err)
	}

	r := bufio.NewReader(httputil.NewChunkedReader(&b))

	payload, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("chunk read: %w", err)
	}

	var push *PushAnswer
	if err := json.Unmarshal(payload, &push); err != nil {
		return nil, fmt.Errorf("push payload json unmarshal: %w", err)
	}

	return push, nil
}

func donePush(opts MinistryOpts, requestID uuid.UUID, eventType string) error {
	logs.Infof("donePush\n")

	cmd := fmt.Sprintf("readpush -id %s -event %s %s", requestID, eventType, opts.token)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, opts.controlIP, cmd)

	if opts.fake {
		logs.Debugf("fake push done\n")

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
		return fmt.Errorf("ssh run: %w", err)
	}

	return nil
}
