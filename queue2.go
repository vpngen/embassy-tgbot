package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/crypto/pbkdf2"
)

const (
	acceptPrefix = "a-"
	rejectPrefix = "r-"
)

const (
	billqPrefix2 = "bq2"
	billqKeyLen2 = 16 - len(billqPrefix2)
	billqSalt2   = "Lewm)Ow6"
)

// Check bill stages
const (
	CkBillStageNone2 = iota
	CkBillStageSend2
	CkBillStageAccept2
	CkBillStageReject2
)

// CkBillQueue2 - queue with bills for manual or auto check.
type CkBillQueue2 struct {
	Stage       int    `json:"stage"`
	CkBillQueue []byte `json:"billq_id"`
}

// PutBill2 - put bill in the queue
func PutBill2(dbase *badger.DB, billqID []byte) ([]byte, error) {
	var key []byte

	bill := &CkBillQueue2{
		Stage:       CkBillStageNone2,
		CkBillQueue: billqID,
	}

	data, err := json.Marshal(bill)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	err = dbase.Update(func(txn *badger.Txn) error {
		key = queueID2(billqID)

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

// SetBill2 - .
func SetBill2(dbase *badger.DB, id []byte, stage int) error {
	err := dbase.Update(func(txn *badger.Txn) error {
		bill := &CkBillQueue2{}

		data, err := getBill2(txn, id)
		if err != nil {
			return fmt.Errorf("get bill: %w", err)
		}

		err = json.Unmarshal(data, bill)
		if err != nil {
			return fmt.Errorf("unmarhal: %w", err)
		}

		bill.Stage = stage

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

func queueID2(id []byte) []byte {
	key := pbkdf2.Key(id, []byte(billqSalt2), 2048, billqKeyLen2, sha256.New)

	return append([]byte(billqPrefix2), key...)
}

// ResetBill2 - .
func ResetBill2(dbase *badger.DB, id []byte) error {
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

// GetBill2 - .
func GetBill2(dbase *badger.DB, id []byte) (*CkBillQueue2, error) {
	bill := &CkBillQueue2{}

	err := dbase.View(func(txn *badger.Txn) error {
		data, err := getBill2(txn, id)
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

func getBill2(txn *badger.Txn, id []byte) ([]byte, error) {
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

/*func qrun2(db *badger.DB, bot2 *tgbotapi.BotAPI, ckChatID int64) {
	key, bill, err := getNextCkBillQueue2(db, CkBillStageNone2)
	if err != nil {
		return
	}

	msg := tgbotapi.NewPhoto(ckChatID, tgbotapi.FileID(bill.FileID))
	// msg.ReplyMarkup = WannabeKeyboard
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ProtectContent = true

	if _, err := bot2.Send(msg); err != nil {
		logs.Errf("send: %w\n", err)

		return
	}

	err = SetBill2(db, key, CkBillStageSend2)
	if err != nil {
		logs.Errf("set billq send2: %w", err)

		return
	}
}*/

func getNextCkBillQueue2(db *badger.DB, stage int) ([]byte, *CkBillQueue2, error) {
	var key []byte

	bill := &CkBillQueue2{}

	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)

		defer it.Close()

		prefix := []byte(billqPrefix2)

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

// SendBill2 - .
func SendBill2(db *badger.DB, bot2 *tgbotapi.BotAPI, billqID []byte, ckChatID int64, data []byte) error {
	id, err := PutBill2(db, billqID)
	if err != nil {
		return fmt.Errorf("put billq2: %w", err)
	}

	photo := tgbotapi.NewPhoto(ckChatID, tgbotapi.FileBytes{Name: "фотка", Bytes: data})
	// msg.ReplyMarkup = WannabeKeyboard
	// msg.ParseMode = tgbotapi.ModeMarkdown
	// photo.Caption =
	photo.ReplyMarkup = makeCheckBillKeyboard(fmt.Sprintf("%x", id))
	photo.ProtectContent = true

	if _, err := bot2.Request(photo); err != nil {
		return fmt.Errorf("request2: %w", err)
	}

	err = SetBill2(db, id, CkBillStageSend2)
	if err != nil {
		return fmt.Errorf("set billq send2: %w", err)
	}

	return nil
}

// makeCheckBillKeyboard - set wanna keyboard.
func makeCheckBillKeyboard(id string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Подтвердить", acceptPrefix+id),
			tgbotapi.NewInlineKeyboardButtonData("Отвергнуть", rejectPrefix+id),
		),
	)
}
