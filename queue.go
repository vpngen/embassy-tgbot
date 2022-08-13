package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v3"
	"golang.org/x/crypto/pbkdf2"
)

const (
	billqPrefix = "billq"
)

// Check bill stages
const (
	CkBillStageNone = iota
	CkBillStageSend
	CkBillStageAccept
	CkBillStageReject
)

// CkBillQueue - queue with bills for manual or auto check.
type CkBillQueue struct {
	Stage        int    `json:"stage"`
	ChatID       int64  `json:"chat_id"`        // user
	FileUniqueID string `json:"file_unique_id"` // photo
	UpdateTime   int64  `json:"updatetime"`
}

// Put - put bill in the queue
func Put(dbase *badger.DB, chatID int64, fileID string) error {
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

	key := queueID(chatID)
	err = dbase.Update(func(txn *badger.Txn) error {
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

	return pbkdf2.Key(binary.BigEndian.AppendUint64([]byte{}, uint64(chatID)), salt, 1024, 16, sha256.New)
}
