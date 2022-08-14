package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v3"
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
	CkBillStageAccept
	CkBillStageReject
)

// ErrGenUniqQueueID - .
var ErrGenUniqQueueID = errors.New("gen uniq id")

// CkBillQueue - queue with bills for manual or auto check.
type CkBillQueue struct {
	Stage        int    `json:"stage"`
	ChatID       int64  `json:"chat_id"`        // user
	FileUniqueID string `json:"file_unique_id"` // photo
	UpdateTime   int64  `json:"updatetime"`
}

// PutBill - put bill in the queue
func PutBill(dbase *badger.DB, chatID int64, fileID string) error {
	bill := &CkBillQueue{
		ChatID:       chatID,
		FileUniqueID: fileID,
		Stage:        CkBillStageNone,
		UpdateTime:   time.Now().Unix(),
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

		e := badger.NewEntry(key, data)
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

	return pbkdf2.Key(binary.BigEndian.AppendUint64([]byte{}, uint64(chatID)), salt, 1024, billqKeyLen, sha256.New)
}

// SetBill - .
func SetBill(dbase *badger.DB, id []byte, stage int) error {
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
