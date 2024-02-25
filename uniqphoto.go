package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
)

const (
	photoPrefix = "photo_"
	photoTTL    = 90 * 24 * time.Hour // 90 days
)

func IsNoUniqPhoto(db *badger.DB, sum []byte) (bool, error) {
	key := append([]byte(photoPrefix), sum[:]...)

	exists := []byte{}

	if err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}

			return fmt.Errorf("get: %w", err)
		}

		err = item.Value(func(v []byte) error {
			exists = append([]byte{}, v...)

			return nil
		})
		if err != nil {
			return fmt.Errorf("value: %w", err)
		}

		return nil
	}); err != nil {
		return false, fmt.Errorf("db: %w", err)
	}

	if len(exists) != 0 {
		return false, nil
	}

	return true, nil
}

func NewUniqPhoto(db *badger.DB, sum []byte) error {
	key := append([]byte(photoPrefix), sum[:]...)

	if err := db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(key, sum[:]).WithTTL(photoTTL)
		if err := txn.SetEntry(e); err != nil {
			return fmt.Errorf("set: %w", err)
		}

		return nil
	}); err != nil {
		return nil
	}

	return nil
}
