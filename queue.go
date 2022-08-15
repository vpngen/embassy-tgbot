package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vpngen/embassy-tgbot/logs"
	"golang.org/x/crypto/pbkdf2"
)

const (
	billqPrefix = "billq"
	billqKeyLen = 16 - len(billqPrefix)
)

// Check bill stages
const (
	CkBillStageNone = iota
	CkBillStageSend
	CkBillStageDesicion
)

// ErrGenUniqQueueID - .
var ErrGenUniqQueueID = errors.New("gen uniq id")

// CkBillQueue - queue with bills for manual or auto check.
type CkBillQueue struct {
	Stage      int    `json:"stage"`
	ChatID     int64  `json:"chat_id"` // user
	FileID     string `json:"file_id"` // photo
	UpdateTime int64  `json:"updatetime"`
	Accept     bool   `json:"accept"`
}

// PutBill - put bill in the queue
func PutBill(dbase *badger.DB, chatID int64, fileID string) error {
	bill := &CkBillQueue{
		ChatID:     chatID,
		FileID:     fileID,
		Stage:      CkBillStageNone,
		UpdateTime: time.Now().Unix(),
	}

	data, err := json.Marshal(bill)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	err = dbase.Update(func(txn *badger.Txn) error {
		var key []byte

		for i := 1; ; i++ {
			key = queueID(chatID)

			_, err := txn.Get(key)
			if err != nil {
				if errors.Is(err, badger.ErrKeyNotFound) {
					break
				}

				return fmt.Errorf("get: %w", err)
			}

			if i > 10 {
				return ErrGenUniqQueueID
			}
		}

		e := badger.NewEntry(key, data).WithTTL(maxSecondsToLive * time.Second)
		if err := txn.SetEntry(e); err != nil {
			return fmt.Errorf("set: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	return nil
}

func queueID(chatID int64) []byte {
	salt := make([]byte, 8)
	rand.Reader.Read(salt)

	key := pbkdf2.Key(binary.BigEndian.AppendUint64([]byte{}, uint64(chatID)), salt, 1024, billqKeyLen, sha256.New)

	return append([]byte(billqPrefix), key...)
}

// SetBill - .
func SetBill(dbase *badger.DB, id []byte, stage int, accept bool) error {
	err := dbase.Update(func(txn *badger.Txn) error {
		bill := &CkBillQueue{}

		data, err := getBill(txn, id)
		if err != nil {
			return fmt.Errorf("get bill: %w", err)
		}

		err = json.Unmarshal(data, bill)
		if err != nil {
			return fmt.Errorf("unmarhal: %w", err)
		}

		bill.Stage = stage
		bill.UpdateTime = time.Now().Unix()

		data, err = json.Marshal(bill)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}

		e := badger.NewEntry(id, data).WithTTL(maxSecondsToLive * time.Second)
		if err := txn.SetEntry(e); err != nil {
			return fmt.Errorf("set: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("set billq: %w", err)
	}

	return nil
}

// ResetBill - .
func ResetBill(dbase *badger.DB, id []byte) error {
	err := dbase.Update(func(txn *badger.Txn) error {
		if err := txn.Delete(id); err != nil {
			return fmt.Errorf("delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("delete billq: %w", err)
	}

	return nil
}

// GetBill - .
func GetBill(dbase *badger.DB, id []byte) (*CkBillQueue, error) {
	bill := &CkBillQueue{}

	err := dbase.View(func(txn *badger.Txn) error {
		data, err := getBill(txn, id)
		if err != nil {
			return fmt.Errorf("get bill: %w", err)
		}

		err = json.Unmarshal(data, bill)
		if err != nil {
			return fmt.Errorf("unmarhal: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get billq: %w", err)
	}

	return bill, nil
}

func getBill(txn *badger.Txn, id []byte) ([]byte, error) {
	var data []byte

	item, err := txn.Get(id)
	if err != nil {
		return nil, fmt.Errorf("get: %w", err)
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

// QRun - .
func QRun(waitGroup *sync.WaitGroup, db *badger.DB, stop <-chan struct{}, bot, bot2 *tgbotapi.BotAPI, ckChatID int64) {
	defer waitGroup.Done()

	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			qrun(db, bot, bot2, ckChatID)
			timer.Reset(100 * time.Millisecond)
		}
	}
}

func qrun(db *badger.DB, bot, bot2 *tgbotapi.BotAPI, ckChatID int64) {
	ok, err := qrunNew(db, bot, bot2, ckChatID)
	if err != nil {
		logs.Errf("qrun: %s\n", err)

		return
	}

	if !ok {
	}
}

func qrunNew(db *badger.DB, bot, bot2 *tgbotapi.BotAPI, ckChatID int64) (bool, error) {
	key, bill, err := getNextCkBillQueue(db, CkBillStageNone)
	if err != nil {
		return false, fmt.Errorf("get next: %w", err)
	}

	if key == nil {
		return false, nil
	}

	url, err := bot.GetFileDirectURL(bill.FileID)
	if err != nil {
		return false, fmt.Errorf("get file: %w", err)
	}

	photo, err := downloadPhoto(url)
	if err != nil {
		return false, fmt.Errorf("dl photo: %w", err)
	}

	err = SendBill2(db, bot2, key, ckChatID, photo)
	if err != nil {
		return false, fmt.Errorf("send bill2: %w", err)
	}

	err = SetBill(db, key, CkBillStageSend, false)
	if err != nil {
		return false, fmt.Errorf("set billq send: %w", err)
	}

	return true, nil
}

func qrunSeen(db *badger.DB, bot, bot2 *tgbotapi.BotAPI, ckChatID int64) (bool, error) {
	key, bill, err := getNextCkBillQueue(db, CkBillStageDesicion)
	if err != nil {
		return false, fmt.Errorf("get next: %w", err)
	}

	if key == nil {
		return false, nil
	}

	ecode := genEcode()

	switch bill.Accept {
	case true:
		newMsg, err := SendMessage(bot, bill.ChatID, 0, "", ecode)
		if err != nil {
			return false, fmt.Errorf("send grant: %w", err)
		}

		err = setSession(db, bill.ChatID, newMsg.MessageID, stageCleanup)
		if err != nil {
			return false, fmt.Errorf("set session end: %w", err)
		}

		err = ResetBill(db, key)
		if err != nil {
			return false, fmt.Errorf("reset billq: %w", err)
		}
	case false:
		newMsg, err := SendMessage(bot, bill.ChatID, 0, "", ecode)
		if err != nil {
			return false, fmt.Errorf("send reject: %w", err)
		}

		err = setSession(db, bill.ChatID, newMsg.MessageID, stageWait4Bill)
		if err != nil {
			return false, fmt.Errorf("set session next: %w", err)
		}

		err = ResetBill(db, key)
		if err != nil {
			return false, fmt.Errorf("reset billq: %w", err)
		}
	}

	return true, nil
}

func getNextCkBillQueue(db *badger.DB, stage int) ([]byte, *CkBillQueue, error) {
	var key []byte

	bill := &CkBillQueue{}

	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)

		defer it.Close()

		prefix := []byte(billqPrefix)

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

			err = json.Unmarshal(data, bill)
			if err != nil {
				return fmt.Errorf("unmarhal: %w", err)
			}

			if bill.Stage != stage {
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

	return key, bill, nil
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
