package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v3"
)

const (
	sessionSalt = "$Rit5"
)

// Session - session.
type Session struct {
	OurMsgID   int   `json:"our_message_id"`
	Stage      int   `json:"stage"`
	UpdateTime int64 `json:"updatetime"`
}

func sessionID(chatID int64) []byte {
	var int64bytes [8]byte

	binary.BigEndian.PutUint64(int64bytes[:], uint64(chatID))

	digest := sha256.Sum256(int64bytes[:])
	id := append([]byte{'s'}, append([]byte(sessionSalt), digest[:]...)...)

	return id
}

func setSession(dbase *badger.DB, chatID int64, msgID int, stage int) error {
	session := &Session{
		OurMsgID:   msgID,
		Stage:      stage,
		UpdateTime: time.Now().Unix(),
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	key := sessionID(chatID)
	err = dbase.Update(func(txn *badger.Txn) error {
		if err := txn.Set(key, data); err != nil {
			return fmt.Errorf("set: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	return nil
}

func checkSession(dbase *badger.DB, chatID int64) (*Session, error) {
	var (
		data    []byte
		session *Session = &Session{Stage: stageNone}
	)

	key := sessionID(chatID)
	err := dbase.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}

			return fmt.Errorf("get: %w", err)
		}

		err = item.Value(func(v []byte) error {
			data = append([]byte{}, v...)

			return nil
		})
		if err != nil {
			return fmt.Errorf("value: %w", err)
		}

		return nil
	})
	if err != nil {
		return session, fmt.Errorf("db: %w", err)
	}

	if data != nil {
		err := json.Unmarshal(data, session)
		if err != nil {
			return session, fmt.Errorf("unmarhal: %w", err)
		}

		return session, nil
	}

	return session, nil
}
