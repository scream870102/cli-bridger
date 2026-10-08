package main

import (
	"cli-bridger/internal/launcher"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

const settingsLimit = 2 * 1024 * 1024

type Settings struct {
	Target        string          `json:"target"`
	Runner        string          `json:"runner"`
	CustomRunner  string          `json:"customRunner"`
	Path          []string        `json:"path"`
	Values        map[string]any  `json:"values"`
	Enabled       map[string]bool `json:"enabled"`
	HasDescriptor bool            `json:"hasDescriptor"`
}

type RestoredSettings struct {
	Settings Settings `json:"settings"`
	Loaded   *Loaded  `json:"loaded,omitempty"`
	Warning  string   `json:"warning,omitempty"`
}

type settingsRecord struct {
	Settings        Settings        `json:"settings"`
	Raw             json.RawMessage `json:"raw,omitempty"`
	DescriptionFile string          `json:"descriptionFile,omitempty"`
}

func (a *App) settingsDBPath() (string, error) {
	if a.settingsPath != "" {
		return a.settingsPath, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(executable), "cli-bridger.db"), nil
}

func settingsRunner(s Settings) string {
	if s.Runner == "custom" {
		return s.CustomRunner
	}
	return s.Runner
}

func (a *App) SaveSettings(s Settings) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	record := settingsRecord{Settings: s}
	if s.HasDescriptor {
		path, prefix, err := launcher.Resolve(s.Target, settingsRunner(s))
		if err != nil {
			return fmt.Errorf("save settings: %w", err)
		}
		if a.descriptor == nil || !strings.EqualFold(path, a.executable) || !reflect.DeepEqual(prefix, a.prefix) {
			return errors.New("save settings: reload the selected CLI description first")
		}
		record.Raw, err = json.Marshal(a.descriptor)
		if err != nil {
			return err
		}
		record.DescriptionFile = a.descriptionFile
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if len(raw) > settingsLimit {
		return errors.New("settings exceed 2 MiB")
	}
	path, err := a.settingsDBPath()
	if err != nil {
		return err
	}
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return fmt.Errorf("open settings DB %q: %w", path, err)
	}
	defer db.Close()
	return db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("settings"))
		if err != nil {
			return err
		}
		return bucket.Put([]byte("last"), raw)
	})
}

// Restore the cached schema without executing third-party code on startup.
func (a *App) LoadSettings() (*RestoredSettings, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.terminal != nil {
		return nil, errors.New("stop the current command first")
	}
	path, err := a.settingsDBPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("open settings DB %q: %w", path, err)
	}
	defer db.Close()
	var record *settingsRecord
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("settings"))
		if bucket == nil {
			return nil
		}
		raw := bucket.Get([]byte("last"))
		if raw == nil {
			return nil
		}
		if len(raw) > settingsLimit {
			return errors.New("saved settings exceed 2 MiB")
		}
		if err := json.Unmarshal(raw, &record); err != nil {
			return fmt.Errorf("invalid saved settings: %w", err)
		}
		if record == nil {
			return errors.New("invalid saved settings: null record")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}
	a.descriptor, a.executable, a.prefix, a.descriptionFile = nil, "", nil, ""
	result := &RestoredSettings{Settings: record.Settings}
	if !record.Settings.HasDescriptor {
		return result, nil
	}
	executable, prefix, err := launcher.Resolve(record.Settings.Target, settingsRunner(record.Settings))
	if err == nil {
		result.Loaded, err = a.acceptDescription(executable, prefix, record.Raw, record.DescriptionFile)
	}
	if err != nil {
		result.Warning = "無法恢復 CLI 規格，已保留上次輸入，請重新讀取規格：" + err.Error()
	}
	return result, nil
}

func (a *App) ResetSettings() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.terminal != nil {
		return errors.New("stop the current command first")
	}
	path, err := a.settingsDBPath()
	if err != nil {
		return err
	}
	// Remove this application's single DB even when its contents are corrupt.
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reset settings: %w", err)
	}
	a.descriptor, a.executable, a.prefix, a.descriptionFile = nil, "", nil, ""
	return nil
}
