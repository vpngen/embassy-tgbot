package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v3"
)

const (
	billqPrefix2 = "bq2"
	billqKeyLen2 = 16 - len(billqPrefix2)
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
	Stage        int
	CkBillQueue  []byte
	FileUniqueID string `json:"file_unique_id"` // photo
}

// PutBill2 - put bill in the queue
func PutBill2(dbase *badger.DB, billqID []byte, fileID string) error {
	bill := &CkBillQueue2{
		Stage:        CkBillStageNone2,
		FileUniqueID: fileID,
		CkBillQueue:  billqID,
	}

	data, err := json.Marshal(bill)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	err = dbase.Update(func(txn *badger.Txn) error {
		var key []byte

		for i := 1; ; i++ {
			key = queueID2()

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

		e := badger.NewEntry(id, data)
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

func queueID2() []byte {
	key := make([]byte, billqKeyLen2)
	rand.Reader.Read(key)

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
