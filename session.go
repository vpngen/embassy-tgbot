package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

const (
	sessionSalt   = "7=Swinee"
	sessionPrefix = "session_"
)

const (
	SessionCommonTTL = 3 * 24 * time.Hour // 3 days
)

const (
	SessionStateNone = iota
	SessionStatePayloadSomething
	SessionStatePayloadBan
	SessionStatePayloadSecondary
	SessionStateBanOnBan // ban on ban
)

// SessionLabel - session label.
type SessionLabel struct {
	ID    uuid.UUID `json:"id"`
	Label string    `json:"label"`
	Time  time.Time `json:"time"`
}

// Session - session.
type Session struct {
	OurMsgID   int          `json:"our_message_id"`
	Stage      int          `json:"stage"`
	UpdateTime int64        `json:"updatetime"`
	State      int          `json:"state"`
	Label      SessionLabel `json:"label,omitempty"`
	StartLabel string       `json:"start_label,omitempty"`
	Payload    []byte       `json:"payload"`
}

func sessionID(secret []byte, chatID int64) []byte {
	var int64bytes [8 + len(sessionSalt)]byte

	binary.BigEndian.PutUint64(int64bytes[:8], uint64(chatID))
	copy(int64bytes[8:], sessionSalt)

	mac := hmac.New(sha256.New, secret)
	mac.Write(int64bytes[:])

	id := append([]byte(sessionPrefix), mac.Sum(nil)...)

	return id
}

func setSession(dbase *badger.DB, secret []byte, label SessionLabel, chatID int64, msgID int, update int64, stage int, state int, payload []byte) error {
	session := &Session{
		OurMsgID:   msgID,
		Stage:      stage,
		State:      state,
		UpdateTime: update,
		Label:      label,
		Payload:    payload,
	}

	fmt.Fprintf(os.Stderr, "[session] TIME:  %s LABEL: %s\n", session.Label.Time, session.Label.Label)

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	key := sessionID(secret, chatID)
	err = dbase.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(key, data).WithTTL(SessionCommonTTL)
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

func checkSession(dbase *badger.DB, secret []byte, chatID int64) (*Session, error) {
	var (
		data    []byte
		session = &Session{}
	)

	key := sessionID(secret, chatID)
	err := dbase.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
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

		if session.StartLabel != "" && session.Label.Label == "" {
			session.Label.Label = session.StartLabel
		}

		return session, nil
	}

	return session, nil
}

func resetSession(dbase *badger.DB, secret []byte, chatID int64) error {
	key := sessionID(secret, chatID)
	err := dbase.Update(func(txn *badger.Txn) error {
		if err := txn.Delete(key); err != nil {
			return fmt.Errorf("delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}

	return nil
}
