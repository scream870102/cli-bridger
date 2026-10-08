package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"cli-bridger/internal/protocol"
	"github.com/UserExistsError/conpty"
)

func TestMergeEnvironment(t *testing.T) {
	base := []string{"Path=C:\\bin", "DEMO=old", "demo=duplicate", "OTHER=keep", "=C:=C:\\work"}
	copyBase := append([]string{}, base...)
	got := mergeEnvironment(base, map[string]string{"Demo": "", "NEW": "中文=a"})
	joined := strings.Join(got, "\n")
	for _, want := range []string{"Path=C:\\bin", "OTHER=keep", "=C:=C:\\work", "Demo=", "NEW=中文=a"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q: %q", want, got)
		}
	}
	if len(got) != 5 || !reflect.DeepEqual(base, copyBase) {
		t.Fatalf("duplicate or mutated base: %q", got)
	}
}

func TestEnvironmentHelper(t *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-2] != "--" {
		return
	}
	value := os.Getenv("BRIDGER_TEST_ENV")
	if os.Args[len(os.Args)-1] == "--cli-bridger-describe" {
		d := protocol.Descriptor{Version: "1", Name: "env helper", Description: value, Root: protocol.Command{ID: "root", Description: "root", Parameters: []protocol.Parameter{
			{ID: "env", Name: "environment", Description: "child environment", Type: "string", Env: "BRIDGER_TEST_ENV"},
			{ID: "required", Name: "required", Description: "normal argv", Type: "string", Flag: "--required", Required: true},
		}}}
		_ = json.NewEncoder(os.Stdout).Encode(d)
	} else {
		fmt.Printf("ENV-BEGIN[%s]ENV-END\r\n", value)
	}
	os.Exit(0)
}

func TestDiscoveryEnvironmentReload(t *testing.T) {
	t.Setenv("BRIDGER_TEST_ENV", "inherited")
	exe, _ := os.Executable()
	a := &App{}
	loaded, err := a.describe(exe, []string{"-test.run=^TestEnvironmentHelper$", "--"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Descriptor.Description != "inherited" {
		t.Fatal("initial discovery did not inherit")
	}
	loaded, err = a.Reload(nil, map[string]any{"env": "覆寫=value"}, map[string]bool{"env": true})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Descriptor.Description != "覆寫=value" {
		t.Fatal("reload did not apply override")
	}
	loaded, err = a.Reload(nil, map[string]any{"env": "ignored"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Descriptor.Description != "inherited" || os.Getenv("BRIDGER_TEST_ENV") != "inherited" {
		t.Fatal("override leaked or disabled did not inherit")
	}
}

func TestPTYEnvironment(t *testing.T) {
	t.Setenv("BRIDGER_TEST_ENV", "parent")
	exe, _ := os.Executable()
	for _, override := range []map[string]string{{"BRIDGER_TEST_ENV": "子程序=a b"}, {}} {
		p, err := conpty.Start(syscall.EscapeArg(exe)+" -test.run=^TestEnvironmentHelper$ -- --env-output", conpty.ConPtyEnv(mergeEnvironment(os.Environ(), override)))
		if err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		done := make(chan struct{})
		go func() {
			defer close(done)
			buf := make([]byte, 8192)
			for {
				n, e := p.Read(buf)
				output.Write(buf[:n])
				if e != nil {
					return
				}
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		code, err := p.Wait(ctx)
		cancel()
		_ = p.Close()
		awaitPTYReader(t, done)
		if err != nil || code != 0 {
			t.Fatalf("exit %d: %v", code, err)
		}
		want := "parent"
		if v, ok := override["BRIDGER_TEST_ENV"]; ok {
			want = v
		}
		if !strings.Contains(output.String(), "ENV-BEGIN["+want+"]ENV-END") {
			t.Fatalf("missing child env: %q", output.String())
		}
		if os.Getenv("BRIDGER_TEST_ENV") != "parent" {
			t.Fatal("parent mutated")
		}
	}
}
