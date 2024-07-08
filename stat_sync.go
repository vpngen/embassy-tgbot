package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vpngen/embassy-tgbot/logs"
	"golang.org/x/crypto/ssh"
)

const (
	LogSyncDuration = 10 * time.Minute
	maxSendAttempts = 5
	maxLogsPerTime  = 20
)

var ErrMaxSendAttemptsExceeded = fmt.Errorf("max send attempts exceeded")

func statSyncLoop(wg *sync.WaitGroup, stop <-chan struct{}, labelStorage *LabelStorage, ministry MinistryOpts) {
	defer wg.Done()

	tm := time.NewTimer(time.Second)

	defer tm.Stop()

	for {
		select {
		case <-tm.C:
			logs.Info("Start logs sync\n")

			if err := labelStorage.Rotate(); err != nil {
				logs.Errf("logs rotate: %s\n", err)
			}

			if err := sendLogs(labelStorage, ministry); err != nil {
				logs.Errf("logs send: %s\n", err)
			}

			tm.Reset(LogSyncDuration)
		case <-stop:
			return
		}
	}
}

func sendLogs(labelStorage *LabelStorage, ministry MinistryOpts) error {
	entries, err := os.ReadDir(labelStorage.logDirname)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	attempts := 0
	count := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasSuffix(filename, completeLabelSuffix) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			logs.Errf("entry info: %s: %s\n", err, filename)

			continue
		}

		if info.Size() == 0 {
			logs.Warningf("empty log: %s\n", filename)

			if err := os.Remove(filename); err != nil {
				logs.Errf("remove empty log: %s\n", err)
			}

			continue
		}

		if err := sendLog(ministry, filename); err != nil {
			attempts++

			logs.Errf("send log: %s\n", err)
		}

		if attempts >= maxSendAttempts {
			return fmt.Errorf("%w: %d", ErrMaxSendAttemptsExceeded, maxSendAttempts)
		}

		count++

		if count >= maxLogsPerTime {
			logs.Warningf("send max logs count: %d\n", count)

			break
		}
	}

	return nil
}

func sendLog(ministry MinistryOpts, name string) error {
	logs.Infof("sending log: %s\n", name)

	f, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	defer f.Close()

	basename := filepath.Base(name)

	cmd := fmt.Sprintf("synclabels -ch %s", basename)

	fmt.Fprintf(os.Stderr, "%s#%s:22 -> %s\n", sshkeyRemoteUsername, ministry.controlIP, cmd)

	if ministry.fake {
		logs.Debugf("fake log send: %s\n", name)

		return nil
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", ministry.controlIP), ministry.sshConfig)
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("ssh session: %w", err)
	}

	defer session.Close()

	var b, e bytes.Buffer

	session.Stdout = &b
	session.Stderr = &e

	LogTag := "tgembass"
	defer func() {
		switch errstr := e.String(); errstr {
		case "":
			fmt.Fprintf(os.Stderr, "%s: SSH Session StdErr: empty\n", LogTag)
		default:
			fmt.Fprintf(os.Stderr, "%s: SSH Session StdErr:\n", LogTag)
			for _, line := range strings.Split(errstr, "\n") {
				fmt.Fprintf(os.Stderr, "%s: | %s\n", LogTag, line)
			}
		}
	}()

	go func() {
		stdin, err := session.StdinPipe()
		if err != nil {
			// return fmt.Errorf("stdin pipe: %w", err)
			return
		}

		defer stdin.Close()

		if _, err := io.Copy(stdin, f); err != nil {
			// return fmt.Errorf("copy: %w", err)
			return
		}
	}()

	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	if err := session.Wait(); err != nil {
		return fmt.Errorf("wait: %w", err)
	}

	if err := os.Remove(name); err != nil {
		return fmt.Errorf("remove: %w", err)
	}

	return nil
}
