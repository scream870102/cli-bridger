package launcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"cli-bridger/internal/protocol"
)

func makeFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("not an executable; resolution must not launch it"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveNativeAndCustom(t *testing.T) {
	dir := t.TempDir()
	runner := makeFile(t, dir, "custom runner.EXE")
	script := makeFile(t, dir, "script name; literal.py")
	exe, args, err := Resolve(runner, "auto")
	if err != nil || exe != runner || len(args) != 0 {
		t.Fatalf("native: %q %q %v", exe, args, err)
	}
	exe, args, err = Resolve(script, runner)
	if err != nil || exe != runner || !reflect.DeepEqual(args, []string{script}) {
		t.Fatalf("custom runner: %q %q %v", exe, args, err)
	}
	// A custom interpreter may consume an extension that auto does not know.
	other := makeFile(t, dir, "input.tool")
	if _, args, err = Resolve(other, runner); err != nil || !reflect.DeepEqual(args, []string{other}) {
		t.Fatalf("custom extension: %q %v", args, err)
	}
}

func TestResolveStandardInterpreters(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows interpreter lookup uses PATHEXT")
	}
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	t.Setenv("PATHEXT", ".EXE")
	for _, name := range []string{"python.exe", "node.exe", "powershell.exe"} {
		makeFile(t, dir, name)
	}
	for _, tc := range []struct {
		name, runner, exe string
		before            []string
	}{
		{"demo.PY", "auto", "python.exe", []string{"-u"}},
		{"demo.js", "auto", "node.exe", nil},
		{"demo.mjs", "auto", "node.exe", nil},
		{"demo.cjs", "auto", "node.exe", nil},
		{"demo.PS1", "auto", "powershell.exe", []string{"-NoProfile", "-File"}},
		{"python.input", "python", "python.exe", []string{"-u"}},
		{"node.input", "node", "node.exe", nil},
		{"powershell.input", "powershell", "powershell.exe", []string{"-NoProfile", "-File"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := makeFile(t, dir, tc.name)
			exe, args, err := Resolve(script, tc.runner)
			want := append(tc.before, script)
			if err != nil || !strings.EqualFold(exe, filepath.Join(dir, tc.exe)) || !reflect.DeepEqual(args, want) {
				t.Fatalf("got %q %q %v; expected %s %q", exe, args, err, tc.exe, want)
			}
		})
	}
	if _, args, err := Resolve("python", "auto"); err != nil || len(args) != 0 {
		t.Fatalf("native PATH lookup: %q %v", args, err)
	}
}

func TestCustomShellInterpreters(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "custom interpreters")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	script := makeFile(t, dir, "script name; literal.ps1")
	for _, name := range []string{"powershell.exe", "pwsh.exe", "PowerShell.EXE"} {
		t.Run(name, func(t *testing.T) {
			runner := makeFile(t, dir, name)
			exe, args, err := Resolve(script, runner)
			want := []string{"-NoProfile", "-File", script}
			if err != nil || exe != runner || !reflect.DeepEqual(args, want) {
				t.Fatalf("got %q %q %v; want %q %q", exe, args, err, runner, want)
			}
		})
	}
	for _, name := range []string{"cmd.exe", "wsl.exe"} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := Resolve(script, makeFile(t, dir, name)); err == nil {
				t.Fatal("command-mode interpreter accepted")
			}
		})
	}
}

func TestResolveRejectsInvalidTargets(t *testing.T) {
	dir := t.TempDir()
	exe := makeFile(t, dir, "runner.exe")
	script := makeFile(t, dir, "valid.py")
	batch := makeFile(t, dir, "unsafe.CMD")
	for _, tc := range []struct{ name, target, runner string }{
		{"empty", "", "auto"},
		{"missing native", filepath.Join(dir, "missing.exe"), "auto"},
		{"missing script", filepath.Join(dir, "missing.py"), exe},
		{"directory", dir, exe},
		{"batch auto", batch, "auto"},
		{"batch custom", batch, exe},
		{"unknown extension", makeFile(t, dir, "file.txt"), "auto"},
		{"runner command string", script, exe + " --unsafe"},
		{"batch runner", script, batch},
		{"missing runner", script, filepath.Join(dir, "missing.exe")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := Resolve(tc.target, tc.runner); err == nil {
				t.Fatal("invalid target accepted")
			}
		})
	}
}

func TestPythonDemoDiscovery(t *testing.T) {
	if _, err := exec.LookPath("python"); err != nil {
		t.Skip("Python is not installed")
	}
	exe, prefix, err := Resolve(filepath.Join("..", "..", "examples", "demo.py"), "auto")
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(exe, append(prefix, "--cli-bridger-describe")...).Output()
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := protocol.Parse(output)
	if err != nil {
		t.Fatalf("Python discovery violates the protocol: %v", err)
	}
	args, err := protocol.BuildArgs(descriptor, []string{"render"}, map[string]any{"steps": 2}, nil)
	if err != nil || !reflect.DeepEqual(args, []string{"render", "--steps=2"}) {
		t.Fatalf("Python descriptor argv: %q %v", args, err)
	}
}
