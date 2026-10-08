package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"cli-bridger/internal/protocol"
	"github.com/UserExistsError/conpty"
	bolt "go.etcd.io/bbolt"
)

func TestToolSettingsSwitchAndRestart(t *testing.T) {
	for _, scripts := range []bool{false, true} {
		t.Run(map[bool]string{false: "executables", true: "scripts"}[scripts], func(t *testing.T) {
			a, first := settingsTestApp(t)
			runner := first.Target
			second := first
			second.Target = filepath.Join(filepath.Dir(first.Target), "second.exe")
			second.Path = []string{"root", "second"}
			second.Values = map[string]any{"zero": float64(42), "false": true, "blank": "second"}
			second.Enabled = map[string]bool{"zero": false, "false": true}
			if scripts {
				first.Target = filepath.Join(filepath.Dir(runner), "first.py")
				second.Target = filepath.Join(filepath.Dir(runner), "second.py")
				first.Runner, second.Runner = "custom", "custom"
				first.CustomRunner, second.CustomRunner = runner, runner
			}
			load := func(app *App, s Settings, raw string) {
				t.Helper()
				executable, prefix := s.Target, []string(nil)
				if scripts {
					executable, prefix = runner, []string{s.Target}
				}
				if _, err := app.acceptDescription(executable, prefix, []byte(raw), s.Target+".cli-bridger.json"); err != nil {
					t.Fatal(err)
				}
			}
			for _, s := range []Settings{first, second} {
				if err := os.WriteFile(s.Target, []byte("not executable"), 0600); err != nil {
					t.Fatal(err)
				}
				load(a, s, settingsTestDescription)
				if err := a.SaveSettings(s); err != nil {
					t.Fatal(err)
				}
			}
			for _, app := range []*App{a, {settingsPath: a.settingsPath}} {
				last, err := app.LoadSettings()
				if err != nil || last == nil || !reflect.DeepEqual(last.Settings, second) {
					t.Fatalf("last tool: %#v, %v", last, err)
				}
				for _, s := range []Settings{first, second, first} {
					fresh := strings.Replace(settingsTestDescription, `"description":"test"`, `"description":"fresh"`, 1)
					load(app, s, fresh)
					current := app.descriptor
					got, err := app.GetToolSettings()
					if err != nil || got == nil || !reflect.DeepEqual(got.Settings, s) || got.Loaded == nil {
						t.Fatalf("tool %q: %#v, %v", s.Target, got, err)
					}
					if app.descriptor != current || got.Loaded.Descriptor.Description == current.Description {
						t.Fatal("cache replaced fresh schema or lost prior schema")
					}
				}
			}
			pending := Settings{Target: "unfinished", HasDescriptor: false}
			if err := a.SaveSettings(pending); err != nil {
				t.Fatal(err)
			}
			for _, s := range []Settings{first, second} {
				load(a, s, settingsTestDescription)
				got, err := a.GetToolSettings()
				if err != nil || got == nil || !reflect.DeepEqual(got.Settings, s) {
					t.Fatalf("pending erased history: %#v, %v", got, err)
				}
			}
			load(a, first, settingsTestDescription)
			current := a.descriptor
			if err := a.ResetSettings(); err != nil {
				t.Fatal(err)
			}
			if a.descriptor != current || a.executable == "" {
				t.Fatal("reset replaced current descriptor")
			}
			for _, s := range []Settings{first, second} {
				load(a, s, settingsTestDescription)
				got, err := a.GetToolSettings()
				if err != nil || (s.Target == first.Target && got != nil) || (s.Target == second.Target && (got == nil || !reflect.DeepEqual(got.Settings, second))) {
					t.Fatalf("reset changed incorrect history: %#v, %v", got, err)
				}
			}
		})
	}
}

func TestToolSettingsLegacyMigration(t *testing.T) {
	for _, pending := range []bool{false, true} {
		t.Run(map[bool]string{false: "loaded", true: "pending"}[pending], func(t *testing.T) {
			a, first := settingsTestApp(t)
			raw, err := json.Marshal(settingsRecord{Settings: first, Raw: json.RawMessage(settingsTestDescription), DescriptionFile: a.descriptionFile})
			if err != nil {
				t.Fatal(err)
			}
			db, err := bolt.Open(a.settingsPath, 0600, nil)
			if err != nil {
				t.Fatal(err)
			}
			err = db.Update(func(tx *bolt.Tx) error {
				bucket, err := tx.CreateBucket([]byte("settings"))
				if err != nil {
					return err
				}
				return bucket.Put([]byte("last"), raw)
			})
			db.Close()
			if err != nil {
				t.Fatal(err)
			}
			if got, err := a.GetToolSettings(); err != nil || got == nil || !reflect.DeepEqual(got.Settings, first) {
				t.Fatalf("legacy lookup: %#v, %v", got, err)
			}
			second := Settings{Target: filepath.Join(filepath.Dir(first.Target), "second.exe"), Runner: "auto", HasDescriptor: !pending}
			if !pending {
				if err := os.WriteFile(second.Target, []byte("not executable"), 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := a.acceptDescription(second.Target, nil, []byte(settingsTestDescription), ""); err != nil {
					t.Fatal(err)
				}
			}
			if err := a.SaveSettings(second); err != nil {
				t.Fatal(err)
			}
			if _, err := a.acceptDescription(strings.ToUpper(first.Target), nil, []byte(settingsTestDescription), ""); err != nil {
				t.Fatal(err)
			}
			if got, err := a.GetToolSettings(); err != nil || got == nil || !reflect.DeepEqual(got.Settings, first) {
				t.Fatalf("migrated case-insensitive history: %#v, %v", got, err)
			}
		})
	}
}

func TestToolSettingsInvalidHistory(t *testing.T) {
	a, settings := settingsTestApp(t)
	if err := a.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	key, err := toolSettingsKey(a.executable, a.prefix)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"{", "null", strings.Repeat("x", settingsLimit+1), `{"settings":{"hasDescriptor":true},"raw":{"version":"invalid"}}`} {
		db, err := bolt.Open(a.settingsPath, 0600, nil)
		if err != nil {
			t.Fatal(err)
		}
		err = db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte("tools")).Put(key, []byte(raw)) })
		db.Close()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := a.GetToolSettings(); err == nil {
			t.Fatal("invalid tool history accepted")
		}
	}
}

func TestToolSettingsResetLegacyAndLast(t *testing.T) {
	for _, sameTool := range []bool{false, true} {
		for _, pending := range []bool{false, true} {
			a, settings := settingsTestApp(t)
			last := settings
			last.HasDescriptor = !pending
			if !sameTool {
				last.Target = filepath.Join(filepath.Dir(settings.Target), "other.exe")
			}
			raw, err := json.Marshal(settingsRecord{Settings: last, Raw: json.RawMessage(settingsTestDescription)})
			if err != nil {
				t.Fatal(err)
			}
			db, err := bolt.Open(a.settingsPath, 0600, nil)
			if err != nil {
				t.Fatal(err)
			}
			err = db.Update(func(tx *bolt.Tx) error {
				bucket, err := tx.CreateBucket([]byte("settings"))
				if err != nil {
					return err
				}
				return bucket.Put([]byte("last"), raw)
			})
			db.Close()
			if err != nil {
				t.Fatal(err)
			}
			if err := a.ResetSettings(); err != nil {
				t.Fatal(err)
			}
			if got, err := a.GetToolSettings(); err != nil || got != nil {
				t.Fatalf("reset resurrected legacy current tool: %#v, %v", got, err)
			}
			restored, err := (&App{settingsPath: a.settingsPath}).LoadSettings()
			if err != nil || (sameTool && restored != nil) || (!sameTool && (restored == nil || !reflect.DeepEqual(restored.Settings, last))) {
				t.Fatalf("reset last record same=%v pending=%v: %#v, %v", sameTool, pending, restored, err)
			}
			if !sameTool && !pending {
				if _, err := a.acceptDescription(last.Target, nil, []byte(settingsTestDescription), ""); err != nil {
					t.Fatal(err)
				}
				if got, err := a.GetToolSettings(); err != nil || got == nil || !reflect.DeepEqual(got.Settings, last) {
					t.Fatalf("reset lost other legacy tool: %#v, %v", got, err)
				}
			}
		}
	}
	a, _ := settingsTestApp(t)
	current := a.descriptor
	if err := a.ResetSettings(); err != nil || a.descriptor != current {
		t.Fatalf("missing DB reset changed descriptor: %v", err)
	}
	a.descriptor = nil
	if err := a.ResetSettings(); err == nil {
		t.Fatal("reset accepted no loaded tool")
	}
}

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
	return a, Settings{Target: target, Runner: "auto", CustomRunner: "自訂", Path: []string{"root"}, Values: map[string]any{"zero": float64(0), "false": false, "unicode": "你好", "blank": ""}, Enabled: map[string]bool{"zero": true, "false": false}, HasDescriptor: true, CustomEnv: []protocol.EnvironmentVariable{{Name: "HTTP_PROXY", Value: "代理"}, {Name: "EMPTY"}}}
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
	if restarted.descriptor == nil || restarted.executable != settings.Target || restarted.descriptionFile != a.descriptionFile {
		t.Fatal("reset lost in-memory descriptor")
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
	if err := a.ResetSettings(); err == nil {
		t.Fatal("reset did not report corrupt DB")
	}
	if raw, err := os.ReadFile(a.settingsPath); err != nil || string(raw) != "corrupt database" {
		t.Fatalf("reset changed corrupt DB: %v", err)
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

func TestPreferencesAndRecordingToggle(t *testing.T) {
	a, settings := settingsTestApp(t)
	if got, err := a.LoadPreferences(); err != nil || got != defaultPreferences() {
		t.Fatalf("defaults: %#v %v", got, err)
	}
	for _, scrollback := range []int{999, 100001} {
		if err := a.SavePreferences(Preferences{RecordSettings: true, Scrollback: scrollback}); err == nil {
			t.Fatalf("accepted scrollback %d", scrollback)
		}
	}
	if err := a.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	off := Preferences{RecordSettings: false, Scrollback: 20000}
	if err := a.SavePreferences(off); err != nil {
		t.Fatal(err)
	}
	changed := settings
	changed.Values = map[string]any{"changed": true}
	if err := a.SaveSettings(changed); err != nil {
		t.Fatal(err)
	}
	restarted := &App{settingsPath: a.settingsPath}
	if got, err := restarted.LoadPreferences(); err != nil || got != off {
		t.Fatalf("preferences restart: %#v %v", got, err)
	}
	if got, err := restarted.LoadSettings(); err != nil || got != nil {
		t.Fatalf("restored while off: %#v %v", got, err)
	}
	if got, err := a.GetToolSettings(); err != nil || got != nil {
		t.Fatalf("tool history while off: %#v %v", got, err)
	}
	if err := restarted.SavePreferences(defaultPreferences()); err != nil {
		t.Fatal(err)
	}
	got, err := restarted.LoadSettings()
	if err != nil || got == nil || !reflect.DeepEqual(got.Settings, settings) {
		t.Fatalf("records not kept or overwritten while off: %#v %v", got, err)
	}
	db, err := bolt.Open(a.settingsPath, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte("preferences")).Put([]byte("app"), []byte(`{"recordSettings":true,"scrollback":5}`)) })
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&App{settingsPath: a.settingsPath}).LoadPreferences(); err == nil {
		t.Fatal("invalid stored preferences accepted")
	}
}
