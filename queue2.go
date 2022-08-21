package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"

	"github.com/dgraph-io/badger/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/vpngen/embassy-tgbot/logs"
)

const (
	acceptReceiptPrefix = "a-"
	rejectReceiptPrefix = "r-"
)

const (
	receiptqPrefix2 = "rcptq2"
	receiptqKeyLen2 = 16 - len(receiptqPrefix2)
	receiptSalt2    = "Lewm)Ow6"
	receiptTTL2     = 48 * time.Hour
)

// Check bill stages
const (
	CkReceiptStageNone2 = iota
	CkReceiptStageSend2
	CkReceiptStageDecision2
)

// CkReceiptQueue2 - queue with receipts for manual or auto check.
type CkReceiptQueue2 struct {
	Stage          int    `json:"stage"`
	ReceiptQueueID []byte `json:"receiptq_id"`
	Accept         bool   `json:"accept"`
}

// NewCkReceiptQueueData2 - .
func NewCkReceiptQueueData2(receiptQID []byte) ([]byte, []byte, error) {
	key := queueID2(receiptQID)

	receipt := &CkReceiptQueue2{
		Stage:          CkReceiptStageNone2,
		ReceiptQueueID: receiptQID,
	}

	data, err := json.Marshal(receipt)
	if err != nil {
		return nil, nil, fmt.Errorf("parse: %w", err)
	}

	return key, data, nil
}

// PutReceipt2 - put receipt in the queue
func PutReceipt2(dbase *badger.DB, receiptQID []byte) ([]byte, error) {
	key, data, err := NewCkReceiptQueueData2(receiptQID)
	if err != nil {
		return nil, fmt.Errorf("new data: %w", err)
	}

	err = dbase.Update(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		if err == nil {
			return nil
		}

		if !errors.Is(err, badger.ErrKeyNotFound) {
			return fmt.Errorf("get: %w", err)
		}

		e := badger.NewEntry(key, data).WithTTL(maxSecondsToLive * time.Second)
		if err := txn.SetEntry(e); err != nil {
			return fmt.Errorf("set: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("update: %w", err)
	}

	return key, nil
}

// UpdateReceipt2 - .
func UpdateReceipt2(dbase *badger.DB, id []byte, stage int, accept bool) error {
	err := dbase.Update(func(txn *badger.Txn) error {
		data, err := getReceipt2(txn, id)
		if err != nil {
			return fmt.Errorf("get receipt: %w", err)
		}

		data, err = updateReceipt2(data, stage, accept)
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
		return fmt.Errorf("set receipt: %w", err)
	}

	return nil
}

func updateReceipt2(data []byte, stage int, accept bool) ([]byte, error) {
	bill := &CkReceiptQueue2{}

	err := json.Unmarshal(data, bill)
	if err != nil {
		return nil, fmt.Errorf("unmarhal: %w", err)
	}

	bill.Stage = stage
	bill.Accept = accept

	data, err = json.Marshal(bill)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	return data, nil
}

func queueID2(id []byte) []byte {
	key := pbkdf2.Key(id, []byte(receiptSalt2), 2048, receiptqKeyLen2, sha256.New)

	return append([]byte(receiptqPrefix2), key...)
}

// ResetReceipt2 - .
func ResetReceipt2(dbase *badger.DB, id []byte) error {
	err := dbase.Update(func(txn *badger.Txn) error {
		if err := txn.Delete(id); err != nil {
			return fmt.Errorf("delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("delete receipt: %w", err)
	}

	return nil
}

func getReceipt2(txn *badger.Txn, id []byte) ([]byte, error) {
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

// QRun2 - .
func QRun2(waitGroup *sync.WaitGroup, db *badger.DB, stop <-chan struct{}, bot, bot2 *tgbotapi.BotAPI, ckChatID int64) {
	defer waitGroup.Done()

	/*timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()*/

	for {
		select {
		case <-stop:
			return
			/*case <-timer.C:
			qrun2(db, bot2, ckChatID)
			timer.Reset(100 * time.Millisecond)*/
		}
	}
}

func qrun2(db *badger.DB, bot2 *tgbotapi.BotAPI, ckChatID int64) {
	key, bill, err := getNextCkReceiptQueue2(db, CkReceiptStageDecision2)
	if err != nil {
		return
	}

	err = UpdateReceipt(db, key, CkReceiptStageReceived, bill.Accept)
	if err != nil {
		logs.Errf("set billq send2: %w", err)

		return
	}
}

func getNextCkReceiptQueue2(db *badger.DB, stage int) ([]byte, *CkReceiptQueue2, error) {
	var key []byte

	bill := &CkReceiptQueue2{}

	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)

		defer it.Close()

		prefix := []byte(receiptqPrefix2)

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

			break
		}

		err := json.Unmarshal(data, bill)
		if err != nil {
			return fmt.Errorf("unmarhal: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get next: %w", err)
	}

	return key, bill, nil
}

// SendReceipt2 - .
func SendReceipt2(db *badger.DB, bot2 *tgbotapi.BotAPI, billqID []byte, ckChatID int64, data []byte) error {
	id, err := PutReceipt2(db, billqID)
	if err != nil {
		return fmt.Errorf("put billq2: %w", err)
	}

	photo := tgbotapi.NewPhoto(ckChatID, tgbotapi.FileBytes{Name: "фотка", Bytes: data})
	// msg.ReplyMarkup = WannabeKeyboard
	// msg.ParseMode = tgbotapi.ModeMarkdown
	// photo.Caption =
	photo.ReplyMarkup = makeCheckReceiptKeyboard(fmt.Sprintf("%x", id))
	photo.ProtectContent = true

	if _, err := bot2.Request(photo); err != nil {
		return fmt.Errorf("request2: %w", err)
	}

	err = UpdateReceipt2(db, id, CkReceiptStageSend2, false)
	if err != nil {
		return fmt.Errorf("set receipt send2: %w", err)
	}

	return nil
}

// makeCheckReceiptKeyboard - set wanna keyboard.
func makeCheckReceiptKeyboard(id string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Подтвердить", acceptReceiptPrefix+id),
			tgbotapi.NewInlineKeyboardButtonData("Отвергнуть", rejectReceiptPrefix+id),
		),
	)
}
