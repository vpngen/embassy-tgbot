package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	currentLabelSuffix  = ".log.current"
	completeLabelSuffix = ".log"
	filePerm            = 0o644
	maxRenameAttempts   = 5
)

var ErrMaxRenameAttemptsExceeded = fmt.Errorf("max rename attempts exceeded")

type LabelStorage struct {
	mu              sync.Mutex
	logname         string
	currentFilename string
	logDirname      string
}

type LabelMap struct {
	UpdateTime  int64            `json:"updatetime"`
	LabelCounts map[string]int64 `json:"labelcounts"`
}

func NewLabelStorage(filename string) (*LabelStorage, error) {
	filename = strings.TrimSuffix(filename, completeLabelSuffix)
	filename = strings.TrimSuffix(filename, currentLabelSuffix)

	if filename == "" {
		return nil, nil
	}

	ls := &LabelStorage{
		logname:         filename,
		currentFilename: filename + currentLabelSuffix,
		logDirname:      filepath.Dir(filename),
	}

	ls.mu.Lock()
	defer ls.mu.Unlock()

	if err := ls.touch(); err != nil {
		return nil, fmt.Errorf("touch: %w", err)
	}

	return ls, nil
}

func (ls *LabelStorage) touch() error {
	if ls.notConfigured() {
		return nil
	}

	w, err := os.OpenFile(ls.currentFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, filePerm)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	defer w.Close()

	return nil
}

func (ls *LabelStorage) notConfigured() bool {
	return ls == nil || ls.logname == ""
}

func (ls *LabelStorage) Update(label SessionLabel) error {
	if ls.notConfigured() {
		return nil
	}

	ls.mu.Lock()
	defer ls.mu.Unlock()

	return ls.updateLabel(label)
}

func (ls *LabelStorage) updateLabel(label SessionLabel) error {
	if ls.notConfigured() {
		return nil
	}

	w, err := os.OpenFile(ls.currentFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, filePerm)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	fmt.Fprintf(w, "%d|%s|%s\n", label.Time.Unix(), label.ID, label.Label)

	defer w.Close()

	return nil
}

func (ls *LabelStorage) Rotate() error {
	if ls.notConfigured() {
		return nil
	}

	ls.mu.Lock()
	defer ls.mu.Unlock()

	return ls.rotate()
}

func (ls *LabelStorage) rotate() error {
	if ls.notConfigured() {
		return nil
	}

	buf := make([]byte, 4)
	now := time.Now().UTC().Format(time.RFC3339)

	for i := 0; i < maxRenameAttempts; i++ {
		binary.BigEndian.PutUint32(buf, uint32(rand.Int31n(math.MaxInt32)))
		filename := fmt.Sprintf("%s-%s-%x%s", ls.logname, now, buf, completeLabelSuffix)

		if _, err := os.Stat(filename); os.IsNotExist(err) {
			if err := os.Rename(ls.currentFilename, filename); err != nil {
				return fmt.Errorf("rename: %s: %s: %w", ls.currentFilename, filename, err)
			}

			ls.touch()

			return nil
		}
	}

	return fmt.Errorf("rotate: %w: %d", ErrMaxRenameAttemptsExceeded, maxRenameAttempts)
}

func setLabel(l SessionLabel) SessionLabel {
	if l.Label == "" && l.Time.IsZero() && l.ID == uuid.Nil {
		label := ""
		x := rand.Intn(len(MainTrackQuizMessage))
		for prefix := range MainTrackQuizMessage {
			if x == 0 {
				label = prefix + label
				if len(label) > 64 {
					label = label[:64]
				}

				break
			}

			x--
		}

		return SessionLabel{
			Label: label,
			Time:  time.Now(),
			ID:    uuid.New(),
		}
	}

	return l
}
