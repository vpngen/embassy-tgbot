package main

import (
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ChatsWins - window per chat limiter.
type ChatsWins struct {
	mu sync.Mutex
	M  map[int64]int
}

// NewChatsWins - constructor.
func NewChatsWins() *ChatsWins {
	return &ChatsWins{
		mu: sync.Mutex{},
		M:  make(map[int64]int),
	}
}

// Get - fetch or create.
func (cw *ChatsWins) Get(chatID int64) int {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	n := cw.M[chatID]
	cw.M[chatID] = n + 1

	return n
}

// Release - remove.
func (cw *ChatsWins) Release(chatID int64) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	delete(cw.M, chatID)
}

// FreeChatWindow - check if chat window is free.
func FreeChatWindow(bot *tgbotapi.BotAPI, cw *ChatsWins, chatID int64, msgID int, ecode string) bool {
	if cw.Get(chatID) > 0 {
		// 0 if callback
		if msgID > 0 {
			// don't handle errs
			removeMsg(bot, chatID, msgID) //nolint
		}

		return false
	}

	return true
}
