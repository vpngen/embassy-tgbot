package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const (
	labelTempSuffix = ".tmp"
	filePerm        = 0o644
)

type LabelStorage struct {
	mu       sync.Mutex
	filename string
}

type LabelMap struct {
	UpdateTime  int64            `json:"updatetime"`
	LabelCounts map[string]int64 `json:"labelcounts"`
}

func NewLabelStorage(filename string) (*LabelStorage, error) {
	ls := &LabelStorage{
		filename: filename,
	}

	w, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}

	defer w.Close()

	return ls, nil
}

func (ls *LabelStorage) Update(label string) error {
	if ls == nil {
		return nil
	}

	ls.mu.Lock()
	defer ls.mu.Unlock()

	return ls.updateLabel(label)
}

func (ls *LabelStorage) updateLabel(label string) error {
	if ls.filename == "" {
		return nil
	}

	tmpFilename := ls.filename + labelTempSuffix

	w, err := os.OpenFile(tmpFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	defer w.Close()

	r, err := os.OpenFile(ls.filename, os.O_RDONLY|os.O_CREATE, filePerm)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	defer r.Close()

	lm := LabelMap{}

	if err := json.NewDecoder(r).Decode(&lm); err != nil &&
		err != io.ErrUnexpectedEOF &&
		err != io.EOF {
		return fmt.Errorf("decode: %w", err)
	}

	if lm.LabelCounts == nil {
		lm.LabelCounts = make(map[string]int64)
	}

	lm.LabelCounts[label]++
	lm.UpdateTime = time.Now().Unix()

	e := json.NewEncoder(w)
	e.SetIndent("", "  ")

	if err := e.Encode(lm); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := w.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	if err := os.Remove(ls.filename); err != nil {
		return fmt.Errorf("remove main: %w", err)
	}

	if err := os.Link(tmpFilename, ls.filename); err != nil {
		return fmt.Errorf("rename temp to name: %w", err)
	}

	if err := os.Remove(tmpFilename); err != nil {
		return fmt.Errorf("remove temp: %w", err)
	}

	return nil
}
