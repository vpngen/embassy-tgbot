package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"

	"github.com/dgraph-io/badger/v4"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/vpngen/embassy-tgbot/logs"
)

const (
	receiptqPrefix = "rcpt1q"
	receiptqKeyLen = 16 - len(receiptqPrefix)
	receiptSalt    = "$WrojOb4"
	receiptTTL     = 48 * time.Hour
)

// Receipt checking stages
const (
	CkReceiptStageNone     = iota
	CkReceiptStageSent     // sent on review service
	CkReceiptStageReceived // decision recieved
)

// ErrKeyConflict - receipt for this chat exists.
var ErrKeyConflict = errors.New("key conflict")

// CkReceipt - receipt for manual or auto check.
type CkReceipt struct {
	Stage    int    `json:"stage"`    // stage
	ChatID   int64  `json:"chat_id"`  // user
	FileID   string `json:"file_id"`  // photo
	Accepted bool   `json:"accepted"` // status
	Reason   int    `json:"reason"`   // rejection reason
}

// PutReceipt - put receipt in the queue.
func PutReceipt(dbase *badger.DB, chatID int64, fileID string) error {
	receipt := &CkReceipt{
		ChatID: chatID,
		FileID: fileID,
		Stage:  CkReceiptStageNone,
	}

	data, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	err = dbase.Update(func(txn *badger.Txn) error {
		var key []byte = queueID(chatID)

		_, err := txn.Get(key)
		if err != nil {
			if !errors.Is(err, badger.ErrKeyNotFound) {
				return fmt.Errorf("get: %w", err)
			}
		}

		/*if err == nil {
			return ErrKeyConflict
		}*/

		e := badger.NewEntry(key, data).WithTTL(receiptTTL)
		if err := txn.SetEntry(e); err != nil {
			return fmt.Errorf("set: %w", err)
		}

		// fmt.Printf("*** q1 id: %x\n", key)

		return nil
	})

	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	return nil
}

func queueID(chatID int64) []byte {
	key := pbkdf2.Key([]byte(
		strconv.FormatInt(chatID, 10)),
		[]byte(receiptSalt),
		2048,
		receiptqKeyLen,
		sha256.New,
	)

	return append([]byte(receiptqPrefix), key...)
}

// UpdateReceipt - update receipt review status and stage.
func UpdateReceipt(dbase *badger.DB, id []byte, stage int, accept bool, reason int) error {
	// fmt.Printf("*** update q1: %x stage=%d\n", id, stage)

	err := dbase.Update(func(txn *badger.Txn) error {
		receipt := &CkReceipt{}

		data, err := getReceipt(txn, id)
		if err != nil {
			return fmt.Errorf("get: %w", err)
		}

		err = json.Unmarshal(data, receipt)
		if err != nil {
			return fmt.Errorf("unmarhal: %w", err)
		}

		receipt.Stage = stage
		receipt.Accepted = accept
		receipt.Reason = reason

		data, err = json.Marshal(receipt)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}

		e := badger.NewEntry(id, data).WithTTL(receiptTTL)
		if err := txn.SetEntry(e); err != nil {
			return fmt.Errorf("set: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("update receipt: %w", err)
	}

	return nil
}

// DeleteReceipt - delete record from receipt queue.
func DeleteReceipt(dbase *badger.DB, id []byte) error {
	err := dbase.Update(func(txn *badger.Txn) error {
		if err := txn.Delete(id); err != nil {
			return fmt.Errorf("delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("receipt: %w", err)
	}

	return nil
}

// GetReceipt - .
func GetReceipt(dbase *badger.DB, id []byte) (*CkReceipt, error) {
	receipt := &CkReceipt{}

	err := dbase.View(func(txn *badger.Txn) error {
		data, err := getReceipt(txn, id)
		if err != nil {
			return fmt.Errorf("get: %w", err)
		}

		err = json.Unmarshal(data, receipt)
		if err != nil {
			return fmt.Errorf("unmarhal: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get receipt: %w", err)
	}

	return receipt, nil
}

// help get data from badgerBD.
func getReceipt(txn *badger.Txn, id []byte) ([]byte, error) {
	var data []byte
	// fmt.Printf("*** get q1: %x\n", id)

	item, err := txn.Get(id)
	if err != nil {
		return nil, fmt.Errorf("txn: %w", err)
	}

	err = item.Value(func(v []byte) error {
		data = append([]byte{}, v...)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("value: %w", err)
	}

	return data, nil
}

// ReceiptQueueLoop - recept queue loop.
func ReceiptQueueLoop(waitGroup *sync.WaitGroup, db *badger.DB, stop <-chan struct{}, bot, bot2 *tgbotapi.BotAPI, ckChatID int64, dept DeptOpts) {
	defer waitGroup.Done()

	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			rqround(db, bot, bot2, ckChatID, dept)
			timer.Reset(100 * time.Millisecond)
		}
	}
}

// do round.
func rqround(db *badger.DB, bot, bot2 *tgbotapi.BotAPI, ckChatID int64, dept DeptOpts) {
	ok, err := catchNewReceipt(db, bot, bot2, ckChatID)
	if err != nil {
		logs.Errf("new receipt: %s\n", err)

		return
	}

	if !ok {
		_, err = catchReviewedReceipt(db, bot, bot2, ckChatID, dept)
		if err != nil {
			logs.Errf("reviewed receipt: %s\n", err)

			return
		}
	}
}

// catch one new receipt.
func catchNewReceipt(db *badger.DB, bot, bot2 *tgbotapi.BotAPI, ckChatID int64) (bool, error) {
	key, receipt, err := catchFirstReceipt(db, CkReceiptStageNone)
	if err != nil {
		return false, fmt.Errorf("get next: %w", err)
	}

	if key == nil {
		return false, nil
	}

	url, err := bot.GetFileDirectURL(receipt.FileID)
	if err != nil {
		return false, fmt.Errorf("get file url: %w", err)
	}

	photo, err := downloadPhoto(url)
	if err != nil {
		return false, fmt.Errorf("download photo: %w", err)
	}

	err = SendReceipt2(db, bot2, key, ckChatID, photo)
	if err != nil {
		return false, fmt.Errorf("send receipt2: %w", err)
	}

	err = UpdateReceipt(db, key, CkReceiptStageSent, false, decisionUnknown)
	if err != nil {
		return false, fmt.Errorf("receipt sent: %w", err)
	}

	return true, nil
}

// catch reviewed receipt
func catchReviewedReceipt(db *badger.DB, bot, bot2 *tgbotapi.BotAPI, ckChatID int64, dept DeptOpts) (bool, error) {
	key, receipt, err := catchFirstReceipt(db, CkReceiptStageReceived)
	if err != nil {
		return false, fmt.Errorf("get next: %w", err)
	}

	if key == nil {
		return false, nil
	}

	// check all dialog conditions.
	session, err := checkSession(db, receipt.ChatID)
	if err != nil {
		return false, fmt.Errorf("check session: %w", err)
	}

	ecode := genEcode()

	switch receipt.Accepted {
	case true:
		if desc, ok := DecisionComments[receipt.Reason]; ok && desc != "" {
			if _, err := SendProtectedMessage(bot, receipt.ChatID, 0, desc, ecode); err != nil {
				if IsForbiddenError(err) {
					DeleteReceipt(db, key)
					setSession(db, session.StartLabel, receipt.ChatID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

					return false, fmt.Errorf("send message: %w", err)
				}

				return false, fmt.Errorf("send message: %w", err)
			}
		}

		if err := DeleteReceipt(db, key); err != nil {
			return false, fmt.Errorf("cleanup: %w", err)
		}

		if err := GetBrigadier(bot, session.StartLabel, receipt.ChatID, ecode, dept); err != nil {
			setSession(db, session.StartLabel, receipt.ChatID, 0, 0, stageMainTrackWaitForBill, SessionStatePayloadSomething, nil)

			if _, err := SendProtectedMessage(bot, receipt.ChatID, 0, MainTrackFailMessage, ecode); err != nil {
				if IsForbiddenError(err) {
					setSession(db, session.StartLabel, receipt.ChatID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

					return false, fmt.Errorf("send message: %w", err)
				}

				return false, fmt.Errorf("send fail message: %w", err)
			}

			return false, fmt.Errorf("creation: %w", err)
		}

		if err := setSession(db, session.StartLabel, receipt.ChatID, 0, 0, stageMainTrackCleanup, SessionStatePayloadSomething, nil); err != nil {
			return false, fmt.Errorf("update session: %w", err)
		}

	case false:
		desc, ok := DecisionComments[receipt.Reason]
		if !ok {
			desc = RejectMessage
		}

		newMsg, err := SendProtectedMessage(bot, receipt.ChatID, 0, desc, ecode)
		if err != nil {
			if IsForbiddenError(err) {
				DeleteReceipt(db, key)
				setSession(db, session.StartLabel, receipt.ChatID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return false, fmt.Errorf("send reject message x: %w", err)
			}

			return false, fmt.Errorf("send reject message y: %w", err)
		}

		switch receipt.Reason {
		case decisionRejectUnacceptable:
			if err = setSession(db, session.StartLabel, receipt.ChatID, 0, 0, stageMainTrackCleanup, SessionStatePayloadBan, nil); err != nil {
				return false, fmt.Errorf("update session: %w", err)
			}
		default:
			if err = setSession(db, session.StartLabel, receipt.ChatID, newMsg.MessageID, int64(newMsg.Date), stageMainTrackWaitForBill, SessionStatePayloadSomething, nil); err != nil {
				return false, fmt.Errorf("update session: %w", err)
			}
		}

		if err := DeleteReceipt(db, key); err != nil {
			return false, fmt.Errorf("cleanup: %w", err)
		}
	}

	return true, nil
}

func catchFirstReceipt(db *badger.DB, stage int) ([]byte, *CkReceipt, error) {
	var key []byte

	receipt := &CkReceipt{}

	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)

		defer it.Close()

		prefix := []byte(receiptqPrefix)

		var data []byte
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key = item.Key()
			err := item.Value(func(v []byte) error {
				data = append([]byte{}, v...)

				return nil
			})
			if err != nil {
				return err
			}

			err = json.Unmarshal(data, receipt)
			if err != nil {
				return fmt.Errorf("unmarhal: %w", err)
			}

			if receipt.Stage != stage {
				key = nil
				data = nil

				continue
			}

			break
		}

		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get next: %w", err)
	}

	return key, receipt, nil
}

func downloadPhoto(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("url get: %w", err)
	}

	defer resp.Body.Close()

	photo, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("url body read: %w", err)
	}

	return photo, nil
}
