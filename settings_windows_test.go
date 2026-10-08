package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/UserExistsError/conpty"
	bolt "go.etcd.io/bbolt"
)

const settingsTestDescription = `{"version":"1","name":"測試","description":"test","root":{"id":"root","description":"root"}}`

func settingsTestApp(t *testing.T) (*App, Settings) {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "third-party.exe")
	// An invalid executable proves restoration cannot invoke discovery.
	if err := os.WriteFile(target, []byte("not an executable"), 0600); err != nil {
		t.Fatal(err)
	}
	a := &App{settingsPath: filepath.Join(dir, "cli-bridger.db")}
	if _, err := a.acceptDescription(target, nil, []byte(settingsTestDescription), target+".cli-bridger.json"); err != nil {
		t.Fatal(err)
	}
	return a, Settings{Target: target, Runner: "auto", CustomRunner: "自訂", Path: []string{"root"}, Values: map[string]any{"zero": float64(0), "false": false, "unicode": "你好", "blank": ""}, Enabled: map[string]bool{"zero": true, "false": false}, HasDescriptor: true}
}

func TestSettingsRoundTripAndReset(t *testing.T) {
	a, settings := settingsTestApp(t)
	if got, err := a.LoadSettings(); err != nil || got != nil {
		t.Fatalf("new DB: %#v, %v", got, err)
	}
	if err := a.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	restarted := &App{settingsPath: a.settingsPath}
	got, err := restarted.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Settings, settings) || got.Loaded == nil || got.Warning != "" {
		t.Fatalf("restore: %#v", got)
	}
	if got.Loaded.DescriptionFile != a.descriptionFile || restarted.descriptor.Name != "測試" {
		t.Fatalf("lost schema/source: %#v", got.Loaded)
	}
	neighbor := filepath.Join(filepath.Dir(a.settingsPath), "unrelated.db")
	if err := os.WriteFile(neighbor, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := restarted.ResetSettings(); err != nil {
		t.Fatal(err)
	}
	if restarted.descriptor != nil || restarted.executable != "" || restarted.descriptionFile != "" {
		t.Fatal("reset retained in-memory state")
	}
	if _, err := os.Stat(neighbor); err != nil {
		t.Fatal("reset touched unrelated DB", err)
	}
	if got, err := restarted.LoadSettings(); err != nil || got != nil {
		t.Fatalf("reset restore: %#v, %v", got, err)
	}
}

func TestSettingsMissingTargetAndPendingInputs(t *testing.T) {
	a, settings := settingsTestApp(t)
	if err := a.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(settings.Target); err != nil {
		t.Fatal(err)
	}
	got, err := a.LoadSettings()
	if err != nil || got.Loaded != nil || got.Warning == "" || !reflect.DeepEqual(got.Settings, settings) {
		t.Fatalf("missing target: %#v, %v", got, err)
	}
	settings.Target = "unfinished path"
	settings.HasDescriptor = false
	if err := a.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	got, err = a.LoadSettings()
	if err != nil || got.Loaded != nil || got.Warning != "" || got.Settings.Target != settings.Target {
		t.Fatalf("pending inputs: %#v, %v", got, err)
	}
}

func TestSettingsRejectsStaleSnapshotAndOversize(t *testing.T) {
	a, settings := settingsTestApp(t)
	other := filepath.Join(filepath.Dir(settings.Target), "other.exe")
	if err := os.WriteFile(other, []byte("unused"), 0600); err != nil {
		t.Fatal(err)
	}
	settings.Target = other
	if err := a.SaveSettings(settings); err == nil {
		t.Fatal("accepted stale schema for different CLI")
	}
	settings.HasDescriptor = false
	settings.Values["large"] = strings.Repeat("x", settingsLimit)
	if err := a.SaveSettings(settings); err == nil {
		t.Fatal("accepted oversized settings")
	}
}

func TestSettingsCorruptionAndReset(t *testing.T) {
	a, _ := settingsTestApp(t)
	if err := os.WriteFile(a.settingsPath, []byte("corrupt database"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.LoadSettings(); err == nil {
		t.Fatal("corruption not reported")
	}
	if err := a.ResetSettings(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(a.settingsPath); !os.IsNotExist(err) {
		t.Fatalf("corrupt DB not removed: %v", err)
	}
}

func TestSettingsInvalidRecordAndSnapshot(t *testing.T) {
	a, settings := settingsTestApp(t)
	for _, raw := range []string{"{", "null"} {
		if err := a.SaveSettings(settings); err != nil {
			t.Fatal(err)
		}
		db, err := bolt.Open(a.settingsPath, 0600, nil)
		if err != nil {
			t.Fatal(err)
		}
		err = db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte("settings")).Put([]byte("last"), []byte(raw)) })
		db.Close()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := a.LoadSettings(); err == nil {
			t.Fatalf("invalid record %q accepted", raw)
		}
	}
	if err := a.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	db, err := bolt.Open(a.settingsPath, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("settings"))
		raw := strings.Replace(string(bucket.Get([]byte("last"))), `"version":"1"`, `"version":"invalid"`, 1)
		return bucket.Put([]byte("last"), []byte(raw))
	})
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.LoadSettings()
	if err != nil || got.Loaded != nil || got.Warning == "" || !reflect.DeepEqual(got.Settings, settings) {
		t.Fatalf("invalid schema: %#v, %v", got, err)
	}
}

func TestSettingsPathLockAndRunningReset(t *testing.T) {
	a, settings := settingsTestApp(t)
	defaultPath, err := (&App{}).settingsDBPath()
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if defaultPath != filepath.Join(filepath.Dir(executable), "cli-bridger.db") {
		t.Fatal(defaultPath)
	}
	if err := a.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	db, err := bolt.Open(a.settingsPath, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if err := a.SaveSettings(settings); err == nil {
		t.Error("lock not reported")
	}
	if time.Since(start) > 3*time.Second {
		t.Error("lock timeout exceeded")
	}
	db.Close()
	a.terminal = new(conpty.ConPty)
	if err := a.ResetSettings(); err == nil {
		t.Fatal("reset during run accepted")
	}
	if _, err := os.Stat(a.settingsPath); err != nil {
		t.Fatal("running reset lost settings", err)
	}
	a.terminal = nil
	a.settingsPath = filepath.Join(t.TempDir(), "missing-directory", "cli-bridger.db")
	if err := a.SaveSettings(settings); err == nil {
		t.Fatal("write error not reported")
	}
}
