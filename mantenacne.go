package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	// maintenanceFullFilename is the name of the file that indicates
	// that the bot is in full maintenance mode.
	maintenanceFullFilename = ".maintenance_full"
	// maintenanceNewFilename is the name of the file that indicates
	// that the bot is in maintenance mode for new users.
	maintenanceNewFilename = ".maintenance_newreg"

	// startCheckMaintenanceFilesDelay is the delay before the first check of maintenance files.
	startCheckMaintenanceFilesDelay = 2 * time.Second
	// startStateReportDelay is the delay before the first report of the maintenance state.
	startStateReportDelay = 4 * time.Second

	// maintenanceFilesCheckInterval is the interval between checks of maintenance files.
	mainenanceFilesCheckInterval = 15 * time.Second

	// stateReportInterval is the interval between reports of the maintenance state.
	stateReportInterval = time.Hour

	// minimumSlotsForNewUsers is the minimum number of slots for new users.
	minimumSlotsForNewUsers = 100

	// minimumSlotsForWork is the minimum number of slots for work.
	minimumSlotsForWork = 0
)

type MantenanceState struct {
	filename string
	text     string
	state    bool
	lastmod  time.Time
}

type Maintenance struct {
	sync.Mutex

	dir string

	full    *MantenanceState
	newregs *MantenanceState
}

func NewMantenanceState(filename string) *MantenanceState {
	return &MantenanceState{
		filename: filename,
	}
}

func NewMantenance(dir string) *Maintenance {
	if dir == "" {
		return nil
	}

	return &Maintenance{
		Mutex: sync.Mutex{},

		dir: dir,

		full:    NewMantenanceState(filepath.Join(dir, maintenanceFullFilename)),
		newregs: NewMantenanceState(filepath.Join(dir, maintenanceNewFilename)),
	}
}

// CheckFiles - check the maintenance files.
// Returns:
// - bool: true if full maintenance mode was changed;
// - bool: true if newregs maintenance mode was changed and not covered by full maintenance mode;
// - string: full maintenance text;
// - string: newregs maintenance text;
// - error: error.
func (m *Maintenance) CheckFiles() (bool, bool, string, string, error) {
	if m == nil {
		return false, false, "", "", nil
	}

	m.Lock()
	defer m.Unlock()

	oldfull := m.full.state
	if err := m.full.checkState(); err != nil {
		return false, false, "", "", fmt.Errorf("full: %w", err)
	}

	oldnewreg := m.newregs.state
	if err := m.newregs.checkState(); err != nil {
		return false, false, "", "", fmt.Errorf("newregs: %w", err)
	}

	switch oldfull {
	case true:
		// all maintenance modes are deactivated.
		if !m.full.state && !m.newregs.state {
			return true, false, "", "", nil
		}

		// full maintenance mode is deactivated
		// but newregs maintenance mode is still active.
		if !m.full.state && m.newregs.state {
			return false, true, "", m.newregs.text, nil
		}

		return false, false, m.full.text, "", nil
	default:
		if m.full.state {
			return true, false, m.full.text, "", nil
		}

		if !m.full.state && (m.newregs.state != oldnewreg) {
			return false, true, "", m.newregs.text, nil
		}
	}

	return false, false, "", "", nil
}

func (m *Maintenance) Check() (string, string) {
	if m == nil {
		return "", ""
	}

	m.Lock()
	defer m.Unlock()

	fmt.Fprintf(os.Stderr, "Check: full=%v newregs=%v\n", m.full.state, m.newregs.state)

	if m.full.state {
		return m.full.text, ""
	}

	if m.newregs.state {
		return "", m.newregs.text
	}

	return "", ""
}

func (ms *MantenanceState) checkState() error {
	if ms == nil {
		return nil
	}

	info, err := os.Stat(ms.filename)
	if err != nil || info.IsDir() {
		if os.IsNotExist(err) || (info != nil && info.IsDir()) {
			if ms.state {
				ms.state = false
				ms.text = ""
				ms.lastmod = time.Time{}
			}

			return nil
		}

		return fmt.Errorf("stat: %w", err)
	}

	if ms.state && (info.ModTime() == ms.lastmod) {
		return nil
	}

	f, err := os.Open(ms.filename)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	defer f.Close()

	// read the file
	buf, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	ms.text = string(buf)
	ms.state = true
	ms.lastmod = info.ModTime()

	return nil
}

func checkMantenance(wg *sync.WaitGroup, stop <-chan struct{}, bot *tgbotapi.BotAPI, chatID int64, mnt *Maintenance) {
	defer wg.Done()

	if mnt == nil {
		return
	}

	timerCheckFiles := time.NewTimer(startCheckMaintenanceFilesDelay)
	defer timerCheckFiles.Stop()

	timerSendState := time.NewTimer(startStateReportDelay)
	defer timerSendState.Stop()

	firstCycle := true

	for {
		select {
		case <-timerCheckFiles.C:
			chFull, chNewreg, full, newreg, err := mnt.CheckFiles()
			if err != nil {
				timerCheckFiles.Reset(mainenanceFilesCheckInterval)

				continue
			}

			fmt.Fprintf(os.Stderr, "checkMantenance: full=%v fullText=%v newreg=%v newregText=%v\n", chFull, chNewreg, full != "", newreg != "")

			if firstCycle {
				firstCycle = false

				timerCheckFiles.Reset(mainenanceFilesCheckInterval)

				continue
			}

			if chFull {
				text := "✅ *Full maintenance mode was deactivated*"
				if full != "" {
					text = fmt.Sprintf("❌ *Full maintenance mode was activated*\n\n%s", full)
				}

				if _, err := SendOpenMessage(bot, chatID, 0, false, text, "fmmchanhed"); err != nil {
					fmt.Fprintf(os.Stderr, "SendOpenMessage: %v\n", err)
				}
			}

			if chNewreg {
				text := "✅ *New users registration maintenance mode was deactivated*"
				if newreg != "" {
					text = fmt.Sprintf("⛔️ *New users registration maintenance mode was activated*\n\n%s", newreg)
				}

				if _, err := SendOpenMessage(bot, chatID, 0, false, text, "nrmmchanhed"); err != nil {
					fmt.Fprintf(os.Stderr, "SendOpenMessage: %v\n", err)
				}
			}

			timerCheckFiles.Reset(mainenanceFilesCheckInterval)
		case <-timerSendState.C:
			stFull, stNewreg := mnt.Check()
			if stFull != "" {
				text := fmt.Sprintf("❌ *Full maintenance mode is still activated*\n\n%s", stFull)

				if _, err := SendOpenMessage(bot, chatID, 0, false, text, "ffm"); err != nil {
					fmt.Fprintf(os.Stderr, "SendOpenMessage: %v\n", err)
				}
			}

			if stFull == "" && stNewreg != "" {
				text := fmt.Sprintf("⛔️ *New users registration maintenance mode is still activated*\n\n%s", stNewreg)

				if _, err := SendOpenMessage(bot, chatID, 0, false, text, "nrmm"); err != nil {
					fmt.Fprintf(os.Stderr, "SendOpenMessage: %v\n", err)
				}
			}

			timerSendState.Reset(stateReportInterval)
		case <-stop:
			return
		}
	}
}

func (m *Maintenance) CheckFree(slots int, domains int) {
	if m == nil {
		return
	}

	m.Lock()
	defer m.Unlock()

	if m.full.state {
		return
	}

	if slots <= minimumSlotsForWork || domains <= minimumSlotsForWork {
		// full maintenance mode is activated.
		if err := os.Symlink(m.full.filename+"_tmpl", m.full.filename); err != nil {
			fmt.Fprintf(os.Stderr, "symlink: %v\n", err)
		}

		fmt.Fprintf(os.Stderr, "full maintenance mode is activated: free_slots=%d free_domains=%d\n", slots, domains)

		return
	}

	if slots <= minimumSlotsForNewUsers || domains <= minimumSlotsForNewUsers {
		// new users registration maintenance mode is activated.
		if err := os.Symlink(m.newregs.filename+"_tmpl", m.newregs.filename); err != nil {
			fmt.Fprintf(os.Stderr, "symlink: %v\n", err)
		}

		fmt.Fprintf(os.Stderr, "new users registration maintenance mode is activated: free_slots=%d free_domains=%d\n", slots, domains)

		return
	}
}
