package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"github.com/vpngen/embassy-tgbot/logs"
)

const (
	receiptqPrefix = "receiptq1_"
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
	Stage    int    `json:"stage"`     // stage
	ChatID   int64  `json:"chat_id"`   // user
	FileID   string `json:"file_id"`   // photo
	Accepted bool   `json:"accepted"`  // status
	Reason   int    `json:"reason"`    // rejection reason
	PhotoSum []byte `json:"photo_sum"` // photo checksum
}

// PutReceipt - put receipt in the queue.
func PutReceipt(dbase *badger.DB, secret []byte, chatID int64, fileID string) error {
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
		var key []byte = queueID()

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

		// fmt.Fprintf(os.Stderr, "[receipt put] chat: %x %d %#v\n", key, chatID, receipt)

		return nil
	})
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	return nil
}

func queueID() []byte {
	key := uuid.New()
	id := append([]byte(receiptqPrefix), key[:]...)

	return id
}

// UpdateReceipt - update receipt review status and stage.
func UpdateReceipt(db *badger.DB, id []byte, stage int, accept bool, reason int, sum []byte) error {
	if len(id) == 0 {
		return nil
	}

	err := db.Update(func(txn *badger.Txn) error {
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
		if sum != nil {
			receipt.PhotoSum = append([]byte{}, sum...)
		}

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
func ReceiptQueueLoop(waitGroup *sync.WaitGroup, db *badger.DB, stop <-chan struct{}, bot, bot2 *tgbotapi.BotAPI, ckChatID int64, dept MinistryOpts, sessionSecret []byte, queue2Secret []byte, mnt *Maintenance) {
	defer waitGroup.Done()

	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	timerDebug := time.NewTimer(time.Second)
	defer timerDebug.Stop()

	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			rqround(db, sessionSecret, queue2Secret, bot, bot2, ckChatID, dept, mnt)

			/*if full, newreg := mnt.Check(); full != "" || newreg != "" {
				timer.Reset(3 * time.Minute)

				if full != "" {
					fmt.Fprintf(os.Stderr, "Receipt queue: checkMantenance: fullMode=%v\n", full != "")

					continue
				}

				fmt.Fprintf(os.Stderr, "Receipt queue: checkMantenance: newregMode=%v\n", newreg != "")

				continue
			}*/

			timer.Reset(3 * time.Second)
		case <-timerDebug.C:
			_, _, countNone, err := catchFirstReceipt(db, CkReceiptStageNone) // debug printing
			if err == nil {
				fmt.Fprintf(os.Stderr, "New receipt queue size: %d\n", countNone)
			}

			_, _, countSent, err := catchFirstReceipt(db, CkReceiptStageSent) // debug printing
			if err == nil {
				fmt.Fprintf(os.Stderr, "Sended receipt queue size: %d\n", countSent)
			}

			_, _, countReceived, err := catchFirstReceipt(db, CkReceiptStageReceived) // debug printing
			if err == nil {
				fmt.Fprintf(os.Stderr, "Approved receipt queue size: %d\n", countReceived)
			}

			timerDebug.Reset(30 * time.Second)
		}
	}
}

// do round.
func rqround(db *badger.DB, sessionSecret []byte, queue2Secret []byte, bot, bot2 *tgbotapi.BotAPI, ckChatID int64, dept MinistryOpts, mnt *Maintenance) {
	ok, err := catchNewReceipt(db, queue2Secret, bot, bot2, ckChatID, mnt)
	if err != nil {
		logs.Errf("new receipt: %s\n", err)

		return
	}

	if !ok {
		_, err = catchReviewedReceipt(db, sessionSecret, bot, dept, mnt)
		if err != nil {
			logs.Errf("reviewed receipt: %s\n", err)

			return
		}
	}
}

// catch one new receipt.
func catchNewReceipt(db *badger.DB, secret []byte, bot, bot2 *tgbotapi.BotAPI, ckChatID int64, mnt *Maintenance) (bool, error) {
	key, receipt, count, err := catchFirstReceipt(db, CkReceiptStageNone)
	if err != nil {
		return false, fmt.Errorf("get next: %w", err)
	}

	// fmt.Printf("*** catchNewReceipt: %x %v\n", key, receipt)

	if key == nil || receipt == nil || receipt.FileID == "" {
		return false, nil
	}

	if full, newreg := mnt.Check(); full != "" || (newreg != "" && count >= 20) {
		fmt.Fprintf(os.Stderr, "Reject new receipt: checkMantenance: %x, full mode: %v\n", key, full != "")
		// fmt.Fprintf(os.Stderr, "Reject new receipt: checkMantenance: key=%x\n", key)
		// fmt.Fprintf(os.Stderr, "Reject new receipt: checkMantenance: %#v\n", receipt)

		fakeSum := sha256.Sum256([]byte("xxx"))
		if err := UpdateReceipt(db, key, CkReceiptStageReceived, false, decisionRejectBusy, fakeSum[:]); err != nil {
			fmt.Fprintf(os.Stderr, "Reject new receipt: update receipt: %s\n", err)

			return false, fmt.Errorf("update receipt: %w", err)
		}

		return true, nil
	}

	url, err := bot.GetFileDirectURL(receipt.FileID)
	if err != nil {
		return false, fmt.Errorf("get file url: %w", err)
	}

	photo, err := downloadPhoto(url)
	if err != nil {
		return false, fmt.Errorf("download photo: %w", err)
	}

	err = SendReceipt2(db, bot2, secret, key, ckChatID, photo)
	if err != nil {
		return false, fmt.Errorf("send receipt2: %w", err)
	}

	sum := sha256.Sum256(photo)

	err = UpdateReceipt(db, key, CkReceiptStageSent, false, decisionUnknown, sum[:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] Update receipt: %s\n", err)

		return false, fmt.Errorf("receipt sent: %w", err)
	}

	return true, nil
}

var (
	ErrFullMaintenanceMode   = errors.New("full maintenance mode")
	ErrNewRegMaintenanceMode = errors.New("newreg maintenance mode")
)

// catch reviewed receipt
func catchReviewedReceipt(db *badger.DB, sessionSecret []byte, bot *tgbotapi.BotAPI, dept MinistryOpts, mnt *Maintenance) (bool, error) {
	key, receipt, count, err := catchFirstReceipt(db, CkReceiptStageReceived)
	if err != nil {
		return false, fmt.Errorf("get next: %w", err)
	}

	if key == nil {
		return false, nil
	}

	// fmt.Fprintf(os.Stderr, "[receipt reviewed] %x %#v\n", key, receipt)

	// check all dialog conditions.
	session, err := checkSession(db, sessionSecret, receipt.ChatID)
	if err != nil {
		return false, fmt.Errorf("check session: %w", err)
	}

	ecode := genEcode()

	switch receipt.Accepted {
	case true:
		if full, newreg := mnt.Check(); full != "" || (newreg != "" && count >= 20) {
			if err := DeleteReceipt(db, key); err != nil {
				return false, fmt.Errorf("cleanup: %w", err)
			}

			if full != "" {
				fmt.Fprintf(os.Stderr, "Receipt queue: checkMantenance: fullMode=%v\n", full != "")

				return false, fmt.Errorf("%w: %d", ErrFullMaintenanceMode, count)
			}

			fmt.Fprintf(os.Stderr, "Receipt queue: checkMantenance: newregMode=%v\n", newreg != "")

			return false, fmt.Errorf("%w: %d", ErrNewRegMaintenanceMode, count)
		}

		sum := receipt.PhotoSum

		if desc, ok := DecisionComments[receipt.Reason]; ok && desc != "" {
			if _, err := SendProtectedMessage(bot, receipt.ChatID, 0, false, desc, ecode); err != nil {
				if IsForbiddenError(err) {
					DeleteReceipt(db, key)
					setSession(db, sessionSecret, session.Label, receipt.ChatID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

					return false, fmt.Errorf("send message: %w", err)
				}

				return false, fmt.Errorf("send message: %w", err)
			}
		}

		if err := DeleteReceipt(db, key); err != nil {
			return false, fmt.Errorf("cleanup: %w", err)
		}

		if err := GetBrigadier(bot, session.Label, receipt.ChatID, ecode, dept, mnt); err != nil {
			setSession(db, sessionSecret, session.Label, receipt.ChatID, 0, 0, stageMainTrackWaitForBill, SessionStatePayloadSomething, nil)

			if _, err := SendProtectedMessage(bot, receipt.ChatID, 0, false, MainTrackFailMessage, ecode); err != nil {
				if IsForbiddenError(err) {
					setSession(db, sessionSecret, session.Label, receipt.ChatID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

					return false, fmt.Errorf("send message: %w", err)
				}

				return false, fmt.Errorf("send fail message: %w", err)
			}

			return false, fmt.Errorf("creation: %w", err)
		}

		if sum != nil {
			if err := NewUniqPhoto(db, sum); err != nil {
				return false, fmt.Errorf("new uniq photo: %w", err)
			}
		}

		if err := setSession(db, sessionSecret, session.Label, receipt.ChatID, 0, 0, stageMainTrackCleanup, SessionStatePayloadSomething, nil); err != nil {
			return false, fmt.Errorf("update session: %w", err)
		}

	case false:
		desc, ok := DecisionComments[receipt.Reason]
		if !ok || desc == "" {
			desc = RejectMessage
		}

		// fmt.Fprintf(os.Stderr, "[receipt reject] %d %s %#v\n", receipt.Reason, desc, receipt)

		newMsg, err := SendProtectedMessage(bot, receipt.ChatID, 0, false, desc, ecode)
		if err != nil {
			if IsForbiddenError(err) {
				DeleteReceipt(db, key)
				setSession(db, sessionSecret, session.Label, receipt.ChatID, 0, 0, stageMainTrackCleanup, SessionStateBanOnBan, nil)

				return false, fmt.Errorf("send reject message x: %w", err)
			}

			return false, fmt.Errorf("send reject message y: %w", err)
		}

		switch receipt.Reason {
		case decisionRejectUnacceptable:
			if err = setSession(db, sessionSecret, session.Label, receipt.ChatID, 0, 0, stageMainTrackCleanup, SessionStatePayloadBan, nil); err != nil {
				return false, fmt.Errorf("update session: %w", err)
			}
		default:
			if err = setSession(db, sessionSecret, session.Label, receipt.ChatID, newMsg.MessageID, int64(newMsg.Date), stageMainTrackWaitForBill, SessionStatePayloadSomething, nil); err != nil {
				return false, fmt.Errorf("update session: %w", err)
			}
		}

		if err := DeleteReceipt(db, key); err != nil {
			return false, fmt.Errorf("cleanup: %w", err)
		}
	}

	return true, nil
}

func catchFirstReceipt(db *badger.DB, stage int) ([]byte, *CkReceipt, int, error) {
	var (
		key   []byte
		count int
	)

	receipt := CkReceipt{}
	buf := CkReceipt{}

	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)

		defer it.Close()

		first := true

		prefix := []byte(receiptqPrefix)

		var data []byte

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				data = append([]byte{}, v...)

				return nil
			})
			if err != nil {
				return err
			}

			err = json.Unmarshal(data, &buf)
			if err != nil {
				return fmt.Errorf("unmarhal: %w", err)
			}

			if receipt.Stage != stage {
				continue
			}

			if first {
				first = false
				receipt = buf
				key = append([]byte{}, k...)
			}

			count++
		}

		return nil
	})
	if err != nil {
		return nil, nil, count, fmt.Errorf("get next: %w", err)
	}

	//if key != nil {
	//	fmt.Fprintf(os.Stderr, "[receipt first] %x %#v\n", key, receipt)
	//}

	return key, &receipt, count, nil
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
