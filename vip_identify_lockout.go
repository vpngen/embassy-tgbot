package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	badger "github.com/dgraph-io/badger/v4"

	"github.com/vpngen/embassy-tgbot/logs"
)

// Progressive lockout for the VIP "identify by name+6-words" flow
// (steps_vip_identify.go), to slow down brute-forcing of an existing
// brigade's name+mnemonics. Keyed by chatID (== Telegram user ID for
// private chats), same convention as blockedKey in push_sync.go.
const (
	vipIdentifyFailPrefix = "vip_id_fail_"
	vipIdentifyLockPrefix = "vip_id_lock_"

	// vipIdentifyFailCounterTTL bounds how long a stale failure history is
	// remembered; long enough that the 24h plateau is meaningful against any
	// realistic brute-force attempt, short enough to eventually forgive.
	vipIdentifyFailCounterTTL = 90 * 24 * time.Hour
)

// VIPIdentifyLockedFallback is used only if the admin-panel flow (stage
// "vip_identify_locked") is unreachable; %d is the hours remaining.
const VIPIdentifyLockedFallback = "Извини, ты уже ошибся при вводе имени и 6 слов, подожди, пожалуйста, %d ч. и попробуй снова."

// vipIdentifyLockDurationForCount maps a 1-based failure count to the
// lockout duration: 1st fail -> 1h, 2nd -> 12h, 3rd and every one after -> 24h.
func vipIdentifyLockDurationForCount(count uint32) time.Duration {
	switch {
	case count <= 1:
		return time.Hour
	case count == 2:
		return 12 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func vipIdentifyFailKey(chatID int64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(chatID))

	return append([]byte(vipIdentifyFailPrefix), b[:]...)
}

func vipIdentifyLockKey(chatID int64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(chatID))

	return append([]byte(vipIdentifyLockPrefix), b[:]...)
}

// vipIdentifyLockRemaining returns how long chatID is still locked out of
// the identify flow, or 0 if it's not currently locked. Fails open (treats
// read errors as "not locked") so a badger hiccup never wedges the flow.
func vipIdentifyLockRemaining(db *badger.DB, chatID int64) time.Duration {
	var remaining time.Duration

	_ = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(vipIdentifyLockKey(chatID))
		if err != nil {
			return nil
		}

		expiresAt := item.ExpiresAt()
		if expiresAt == 0 {
			return nil
		}

		remaining = time.Until(time.Unix(int64(expiresAt), 0))
		if remaining < 0 {
			remaining = 0
		}

		return nil
	})

	return remaining
}

// vipIdentifyLockRemainingHours is vipIdentifyLockRemaining rounded up to
// whole hours for display, minimum 1 while still locked.
func vipIdentifyLockRemainingHours(db *badger.DB, chatID int64) int {
	remaining := vipIdentifyLockRemaining(db, chatID)
	if remaining <= 0 {
		return 0
	}

	hours := int(math.Ceil(remaining.Hours()))
	if hours < 1 {
		hours = 1
	}

	return hours
}

// recordVIPIdentifyFailure bumps the persistent failure count for chatID and
// (re)arms the lockout with the escalated duration. Call only for a genuine
// wrong name/words result - not for transport/ministry errors, which aren't
// the user's fault.
func recordVIPIdentifyFailure(db *badger.DB, chatID int64) {
	err := db.Update(func(txn *badger.Txn) error {
		count := uint32(0)

		item, err := txn.Get(vipIdentifyFailKey(chatID))
		switch {
		case err == nil:
			if verr := item.Value(func(v []byte) error {
				if len(v) == 4 {
					count = binary.BigEndian.Uint32(v)
				}
				return nil
			}); verr != nil {
				return verr
			}
		case errors.Is(err, badger.ErrKeyNotFound):
			// first failure on record - count stays 0, becomes 1 below.
		default:
			return err
		}

		count++

		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], count)

		if err := txn.SetEntry(badger.NewEntry(vipIdentifyFailKey(chatID), buf[:]).WithTTL(vipIdentifyFailCounterTTL)); err != nil {
			return err
		}

		lockDuration := vipIdentifyLockDurationForCount(count)

		return txn.SetEntry(badger.NewEntry(vipIdentifyLockKey(chatID), buf[:]).WithTTL(lockDuration))
	})
	if err != nil {
		logs.Errf("recordVIPIdentifyFailure %d: %s\n", chatID, err)
	}
}

// checkVIPIdentifyLockout sends the "please wait" notice and returns true if
// chatID is currently locked out. Callers should stop processing when true.
func checkVIPIdentifyLockout(ctx *StepContext) bool {
	hours := vipIdentifyLockRemainingHours(ctx.Opts.db, ctx.ChatID)
	if hours <= 0 {
		return false
	}

	text := fmt.Sprintf(ctx.FlowVipMessage("vip_identify_locked", VIPIdentifyLockedFallback), hours)

	if _, err := ctx.SendPlain(text); err != nil {
		logs.Errf("[!:%s] send vip_identify_locked: %s\n", ctx.Ecode, err)
	}

	return true
}
