package main

import (
	"cli-bridger/internal/launcher"
	"cli-bridger/internal/protocol"
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
	// Per tool, like Values; validated against the schema only when used.
	CustomEnv []protocol.EnvironmentVariable `json:"customEnv,omitempty"`
}

// Preferences apply to the whole app, unlike per-tool Settings.
type Preferences struct {
	RecordSettings bool `json:"recordSettings"`
	Scrollback     int  `json:"scrollback"`
}

func defaultPreferences() Preferences { return Preferences{RecordSettings: true, Scrollback: 5000} }

func (p Preferences) validate() error {
	if p.Scrollback < 1000 || p.Scrollback > 100000 {
		return errors.New("scrollback must be between 1000 and 100000 lines")
	}
	return nil
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

func toolSettingsKey(executable string, prefix []string) ([]byte, error) {
	if len(prefix) > 0 {
		executable = prefix[len(prefix)-1]
	}
	path, err := filepath.Abs(executable)
	return []byte(strings.ToLower(path)), err
}

func savedToolSettingsKey(s Settings) ([]byte, error) {
	executable, prefix, err := launcher.Resolve(s.Target, settingsRunner(s))
	if err != nil {
		executable, prefix = s.Target, nil
	}
	return toolSettingsKey(executable, prefix)
}

func migrateToolSettings(tx *bolt.Tx) (*bolt.Bucket, error) {
	if history := tx.Bucket([]byte("tools")); history != nil {
		return history, nil
	}
	history, err := tx.CreateBucket([]byte("tools"))
	if err != nil {
		return nil, err
	}
	// Preserve the old single-tool cache before a pending input replaces last.
	if bucket := tx.Bucket([]byte("settings")); bucket != nil {
		previous, err := readSettingsRecord(bucket.Get([]byte("last")))
		if err != nil {
			return nil, err
		}
		if previous != nil && previous.Settings.HasDescriptor {
			key, err := savedToolSettingsKey(previous.Settings)
			if err != nil {
				return nil, err
			}
			if err := history.Put(key, bucket.Get([]byte("last"))); err != nil {
				return nil, err
			}
		}
	}
	return history, nil
}

// Caller holds mu. A missing DB or record means defaults.
func (a *App) preferences() (Preferences, error) {
	if a.prefs != nil {
		return *a.prefs, nil
	}
	prefs := defaultPreferences()
	path, err := a.settingsDBPath()
	if err != nil {
		return prefs, err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		a.prefs = &prefs
		return prefs, nil
	} else if err != nil {
		return prefs, err
	}
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second, ReadOnly: true})
	if err != nil {
		return prefs, fmt.Errorf("open settings DB %q: %w", path, err)
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("preferences"))
		if bucket == nil || bucket.Get([]byte("app")) == nil {
			return nil
		}
		if err := json.Unmarshal(bucket.Get([]byte("app")), &prefs); err != nil {
			return fmt.Errorf("invalid saved preferences: %w", err)
		}
		return prefs.validate()
	})
	if err != nil {
		return defaultPreferences(), err
	}
	a.prefs = &prefs
	return prefs, nil
}

func (a *App) LoadPreferences() (Preferences, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.preferences()
}

// Turning recording off keeps existing tool records; reset still deletes them.
func (a *App) SavePreferences(p Preferences) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := p.validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return err
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
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("preferences"))
		if err != nil {
			return err
		}
		return bucket.Put([]byte("app"), raw)
	})
	if err == nil {
		a.prefs = &p
	}
	return err
}

func (a *App) recording() (bool, error) {
	prefs, err := a.preferences()
	return prefs.RecordSettings, err
}

func (a *App) SaveSettings(s Settings) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if record, err := a.recording(); !record || err != nil {
		return err
	}
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
		history, err := migrateToolSettings(tx)
		if err != nil {
			return err
		}
		if s.HasDescriptor {
			key, err := toolSettingsKey(a.executable, a.prefix)
			if err != nil {
				return err
			}
			if err := history.Put(key, raw); err != nil {
				return err
			}
		}
		return bucket.Put([]byte("last"), raw)
	})
}

func readSettingsRecord(raw []byte) (*settingsRecord, error) {
	if raw == nil {
		return nil, nil
	}
	if len(raw) > settingsLimit {
		return nil, errors.New("saved settings exceed 2 MiB")
	}
	var record *settingsRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, fmt.Errorf("invalid saved settings: %w", err)
	}
	if record == nil {
		return nil, errors.New("invalid saved settings: null record")
	}
	return record, nil
}

// Read the selected tool's history without replacing its freshly loaded schema.
func (a *App) GetToolSettings() (*RestoredSettings, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.descriptor == nil {
		return nil, errors.New("load a CLI description first")
	}
	if record, err := a.recording(); !record || err != nil {
		return nil, err
	}
	key, err := toolSettingsKey(a.executable, a.prefix)
	if err != nil {
		return nil, err
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
		if bucket := tx.Bucket([]byte("tools")); bucket != nil {
			record, err = readSettingsRecord(bucket.Get(key))
			return err
		}
		// Existing installations can read their legacy cache before the first save.
		if bucket := tx.Bucket([]byte("settings")); bucket != nil {
			record, err = readSettingsRecord(bucket.Get([]byte("last")))
			if err != nil || record == nil || !record.Settings.HasDescriptor {
				return err
			}
			executable, prefix, resolveErr := launcher.Resolve(record.Settings.Target, settingsRunner(record.Settings))
			if resolveErr != nil {
				record = nil
				return nil
			}
			previousKey, keyErr := toolSettingsKey(executable, prefix)
			if keyErr != nil {
				return keyErr
			}
			if string(previousKey) != string(key) {
				record = nil
			}
		}
		return nil
	})
	if err != nil || record == nil || !record.Settings.HasDescriptor {
		return nil, err
	}
	// Parsing on a temporary App reuses validation without mutating current state.
	loaded, err := (&App{}).acceptDescription(a.executable, a.prefix, record.Raw, record.DescriptionFile)
	if err != nil {
		return nil, fmt.Errorf("invalid saved CLI description: %w", err)
	}
	return &RestoredSettings{Settings: record.Settings, Loaded: loaded}, nil
}

// Restore the cached schema without executing third-party code on startup.
func (a *App) LoadSettings() (*RestoredSettings, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.terminal != nil {
		return nil, errors.New("stop the current command first")
	}
	if record, err := a.recording(); !record || err != nil {
		return nil, err
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
		record, err = readSettingsRecord(bucket.Get([]byte("last")))
		return err
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
	if a.descriptor == nil {
		return errors.New("load a CLI description first")
	}
	key, err := toolSettingsKey(a.executable, a.prefix)
	if err != nil {
		return err
	}
	path, err := a.settingsDBPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return fmt.Errorf("open settings DB %q: %w", path, err)
	}
	defer db.Close()
	return db.Update(func(tx *bolt.Tx) error {
		history, err := migrateToolSettings(tx)
		if err != nil {
			return err
		}
		if err := history.Delete(key); err != nil {
			return err
		}
		if bucket := tx.Bucket([]byte("settings")); bucket != nil {
			last, err := readSettingsRecord(bucket.Get([]byte("last")))
			if err != nil {
				return err
			}
			if last != nil {
				lastKey, err := savedToolSettingsKey(last.Settings)
				if err != nil {
					return err
				}
				if string(lastKey) == string(key) {
					return bucket.Delete([]byte("last"))
				}
			}
		}
		return nil
	})
}
