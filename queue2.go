package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"github.com/vpngen/embassy-tgbot/logs"
)

const (
	receiptqPrefix2     = "receiptq2_" // <= 16
	maxReceiptsButtoLen = 64
	receiptTTL2         = 48 * time.Hour
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
	PhotoSum       []byte `json:"photo_sum,omitempty"`
}

// PutReceipt2 - put receipt in the queue
func PutReceipt2(dbase *badger.DB, secret []byte, receiptQID []byte) ([]byte, error) {
	key := queueID2()

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

	// fmt.Fprintf(os.Stderr, "[receipt put 2] put: %x %#v\n", key, receipt)

	return key, nil
}

// UpdateReceipt2 - .
func UpdateReceipt2(dbase *badger.DB, key []byte, stage int, accept bool, reason int, sum []byte) error {
	fmt.Fprintf(os.Stderr, "*** UpdateReceipt2: %s\n", string(key))

	if len(key) == 0 {
		return nil
	}

	err := dbase.Update(func(txn *badger.Txn) error {
		data, err := getReceipt2(txn, key)
		if err != nil {
			return fmt.Errorf("get receipt: %w", err)
		}

		data, err = updateReceipt2(data, stage, accept, reason, sum)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}

		e := badger.NewEntry(key, data).WithTTL(receiptTTL2)
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

func updateReceipt2(data []byte, stage int, accept bool, reason int, sum []byte) ([]byte, error) {
	receipt := &CkReceipt2{}

	err := json.Unmarshal(data, receipt)
	if err != nil {
		return nil, fmt.Errorf("unmarhal: %w", err)
	}

	receipt.Stage = stage
	receipt.Accept = accept
	receipt.Reason = reason
	if sum != nil {
		receipt.PhotoSum = sum
	}

	// fmt.Fprintf(os.Stderr, "[receipt update 2] update: %#v\n", receipt)

	data, err = json.Marshal(receipt)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	return data, nil
}

func queueID2() []byte {
	key := uuid.New()

	id := append([]byte(receiptqPrefix2), key[:]...)

	return id
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

	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			qround2(db)
			timer.Reset(3 * time.Second)
		}
	}
}

func qround2(db *badger.DB) {
	fmt.Fprintf(os.Stderr, "*** qround2\n")

	key, receipt, count, err := catchFirstReceipt2(db, CkReceiptStageDecision2)
	if err != nil || key == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "*** qround2: len: %d: %x\n", count, key)

	if err := UpdateReceipt(db, receipt.ReceiptQueueID, CkReceiptStageReceived, receipt.Accept, receipt.Reason, receipt.PhotoSum); err != nil {
		logs.Errf("update receipt: %s: %x\n", err, receipt.ReceiptQueueID)

		// return
	}

	if err := DeleteReceipt2(db, key); err != nil {
		logs.Errf("delete receipt2: %s\n", err)

		return
	}
}

func catchFirstReceipt2(db *badger.DB, stage int) ([]byte, *CkReceipt2, int, error) {
	var (
		key   []byte
		count int
	)

	receipt := CkReceipt2{}
	buf := CkReceipt2{}

	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)

		defer it.Close()

		prefix := []byte(receiptqPrefix2)

		first := true

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

			if buf.Stage != stage {
				continue
			}

			if first {
				first = false
				receipt = buf
				key = make([]byte, len(k))

				copy(key, k)
			}

			count++
		}

		return nil
	})
	if err != nil {
		return nil, nil, 0, fmt.Errorf("get next: %w", err)
	}

	return key, &receipt, count, nil
}

// SendReceipt2 - .
func SendReceipt2(db *badger.DB, bot2 *tgbotapi.BotAPI, secret []byte, receiptQID []byte, ckChatID int64, data []byte) error {
	id, err := PutReceipt2(db, secret, receiptQID)
	if err != nil {
		return fmt.Errorf("put receipt2: %w", err)
	}

	photo := tgbotapi.NewPhoto(ckChatID, tgbotapi.FileBytes{Name: "фотка", Bytes: data})
	// msg.ReplyMarkup = WannabeKeyboard
	// photo.Caption =

	// Ingmund: 24.02.2024
	// photo.ReplyMarkup = makeCheckReceiptKeyboard(fmt.Sprintf("%x", id))
	photo.ParseMode = tgbotapi.ModeMarkdown

	sum := sha256.Sum256(data)

	ok, err := IsNoUniqPhoto(db, sum[:])
	if err != nil {
		return fmt.Errorf("uniq photo: %w", err)
	}

	reason := decisionRejectUnverifiable
	decision := false

	switch ok {
	case true:
		reason = decisionAcceptGeneral
		photo.Caption = fmt.Sprintf(
			"\U00002705"+` *Accept receipt*`+"\nAction: %s\nAction date: *%s*\nBy bot self",
			tgbotapi.EscapeText(tgbotapi.ModeMarkdown, buttons[reason]),
			tgbotapi.EscapeText(tgbotapi.ModeMarkdown, time.Now().Format(time.RFC3339)),
		)

		decision = true
	default:
		reason = decisionRejectDoubled
		photo.Caption = fmt.Sprintf(
			"\U0000274C"+` *Reject receipt*`+"\nAction: %s\nAction date: *%s*\nBy bot self",
			tgbotapi.EscapeText(tgbotapi.ModeMarkdown, buttons[reason]),
			tgbotapi.EscapeText(tgbotapi.ModeMarkdown, time.Now().Format(time.RFC3339)),
		)
	}

	// photo.ProtectContent = true // Oleg Basisty request

	if _, err := bot2.Request(photo); err != nil {
		// return fmt.Errorf("request2: %w", err)
		logs.Errf("request2: %s\n", err)
	}

	// Ingmund: 24.02.2024
	// err = UpdateReceipt2(db, id, CkReceiptStageSend2, false, decisionUnknown)
	fmt.Fprintf(os.Stderr, "*** UpdateReceipt2: %x\n", id)
	err = UpdateReceipt2(db, id, CkReceiptStageDecision2, decision, reason, sum[:])
	if err != nil {
		return fmt.Errorf("update receipt send2: %w", err)
	}

	return nil
}
