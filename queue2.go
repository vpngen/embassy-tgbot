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
	receiptqPrefix2 = "rcpt2q"
	receiptqKeyLen2 = 16 - len(receiptqPrefix2)
	receiptSalt2    = "Lewm)Ow6"
	receiptTTL2     = 48 * time.Hour
)

// Check receipt stages
const (
	CkReceiptStageNone2 = iota
	CkReceiptStageSend2
	CkReceiptStageDecision2
)

// CkReceipt2 - queue with receipts for manual or auto check.
type CkReceipt2 struct {
	Stage          int    `json:"stage"`
	ReceiptQueueID []byte `json:"receiptq_id"`
	Accept         bool   `json:"accept"`
	Reason         int    `json:"reason"` // rejection reason
}

// PutReceipt2 - put receipt in the queue
func PutReceipt2(dbase *badger.DB, receiptQID []byte) ([]byte, error) {
	key := queueID2(receiptQID)

	receipt := &CkReceipt2{
		Stage:          CkReceiptStageNone2,
		ReceiptQueueID: receiptQID,
	}

	data, err := json.Marshal(receipt)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	err = dbase.Update(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		if err == nil {
			return nil
		}

		if !errors.Is(err, badger.ErrKeyNotFound) {
			return fmt.Errorf("get: %w", err)
		}

		e := badger.NewEntry(key, data).WithTTL(receiptTTL2)
		if err := txn.SetEntry(e); err != nil {
			return fmt.Errorf("set: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("update: %w", err)
	}

	// fmt.Printf("*** q2 id (q1 id): %x (%x)\n", key, receiptQID)

	return key, nil
}

// UpdateReceipt2 - .
func UpdateReceipt2(dbase *badger.DB, id []byte, stage int, accept bool, reason int) error {
	// fmt.Printf("*** update q2: %x stage=%d\n", id, stage)

	err := dbase.Update(func(txn *badger.Txn) error {
		data, err := getReceipt2(txn, id)
		if err != nil {
			return fmt.Errorf("get receipt: %w", err)
		}

		data, err = updateReceipt2(data, stage, accept, reason)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}

		e := badger.NewEntry(id, data).WithTTL(receiptTTL2)
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

func updateReceipt2(data []byte, stage int, accept bool, reason int) ([]byte, error) {
	receipt := &CkReceipt2{}

	err := json.Unmarshal(data, receipt)
	if err != nil {
		return nil, fmt.Errorf("unmarhal: %w", err)
	}

	receipt.Stage = stage
	receipt.Accept = accept
	receipt.Reason = reason

	data, err = json.Marshal(receipt)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	return data, nil
}

func queueID2(id []byte) []byte {
	key := pbkdf2.Key(id, []byte(receiptSalt2), 2048, receiptqKeyLen2, sha256.New)

	return append([]byte(receiptqPrefix2), key...)
}

// DeleteReceipt2 - .
func DeleteReceipt2(dbase *badger.DB, id []byte) error {
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

	// fmt.Printf("*** get q2: %x\n", id)

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

// ReceiptQueueLoop2 - .
func ReceiptQueueLoop2(waitGroup *sync.WaitGroup, db *badger.DB, stop <-chan struct{}, bot, bot2 *tgbotapi.BotAPI, ckChatID int64) {
	defer waitGroup.Done()

	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			qround2(db, bot2, ckChatID)
			timer.Reset(100 * time.Millisecond)
		}
	}
}

func qround2(db *badger.DB, bot2 *tgbotapi.BotAPI, ckChatID int64) {
	key, receipt, err := catchFirstReceipt2(db, CkReceiptStageDecision2)
	if err != nil || key == nil {
		return
	}

	// fmt.Printf("*** qround2 rcpt:%x %v\n", key, receipt)

	if err := UpdateReceipt(db, receipt.ReceiptQueueID, CkReceiptStageReceived, receipt.Accept, receipt.Reason); err != nil {
		logs.Errf("update receipt2: %s", err)

		return
	}

	if err := DeleteReceipt2(db, key); err != nil {
		logs.Errf("delete receipt2: %s", err)

		return
	}
}

func catchFirstReceipt2(db *badger.DB, stage int) ([]byte, *CkReceipt2, error) {
	var key []byte

	receipt := &CkReceipt2{}

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

		err := json.Unmarshal(data, receipt)
		if err != nil {
			return fmt.Errorf("unmarhal: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get next: %w", err)
	}

	return key, receipt, nil
}

// SendReceipt2 - .
func SendReceipt2(db *badger.DB, bot2 *tgbotapi.BotAPI, receiptQID []byte, ckChatID int64, data []byte) error {
	id, err := PutReceipt2(db, receiptQID)
	if err != nil {
		return fmt.Errorf("put receipt2: %w", err)
	}

	photo := tgbotapi.NewPhoto(ckChatID, tgbotapi.FileBytes{Name: "фотка", Bytes: data})
	// msg.ReplyMarkup = WannabeKeyboard
	// msg.ParseMode = tgbotapi.ModeMarkdown
	// photo.Caption =
	photo.ReplyMarkup = makeCheckReceiptKeyboard(fmt.Sprintf("%x", id))
	// photo.ProtectContent = true // Oleg Basisty request

	if _, err := bot2.Request(photo); err != nil {
		return fmt.Errorf("request2: %w", err)
	}

	err = UpdateReceipt2(db, id, CkReceiptStageSend2, false, decisionUnknown)
	if err != nil {
		return fmt.Errorf("update receipt send2: %w", err)
	}

	return nil
}
