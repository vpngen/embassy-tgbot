package main

import (
	"fmt"
	"os"
	"sync"
)

const (
	labelTempSuffix = ".tmp"
	filePerm        = 0o644
)

type LabelStorage struct {
	mu      sync.Mutex
	logname string
}

type LabelMap struct {
	UpdateTime  int64            `json:"updatetime"`
	LabelCounts map[string]int64 `json:"labelcounts"`
}

func NewLabelStorage(filename string) (*LabelStorage, error) {
	ls := &LabelStorage{
		logname: filename,
	}

	if ls.logname == "" {
		return ls, nil
	}

	w, err := os.OpenFile(ls.logname, os.O_APPEND|os.O_WRONLY|os.O_CREATE, filePerm)
	if err != nil {
		return nil, fmt.Errorf("create json file: %w", err)
	}

	defer w.Close()

	return ls, nil
}

func (ls *LabelStorage) Update(label SessionLabel) error {
	if ls == nil {
		return nil
	}

	ls.mu.Lock()
	defer ls.mu.Unlock()

	return ls.updateLabel(label)
}

func (ls *LabelStorage) updateLabel(label SessionLabel) error {
	if ls.logname == "" {
		return nil
	}

	w, err := os.OpenFile(ls.logname, os.O_APPEND|os.O_WRONLY|os.O_CREATE, filePerm)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	fmt.Fprintf(w, "%d|%s|%s\n", label.Time.Unix(), label.ID, label.Label)

	defer w.Close()

	return nil
}
