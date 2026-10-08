// Package launcher resolves a program or script into a native executable and
// literal argument prefix. It never parses a command line or invokes a shell.
package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Resolve supports auto, python, node, powershell, or a custom native runner.
// The caller appends discovery or execution arguments to prefix.
func Resolve(target, runner string) (executable string, prefix []string, err error) {
	if strings.TrimSpace(target) == "" {
		return "", nil, fmt.Errorf("choose an executable or script")
	}
	ext := strings.ToLower(filepath.Ext(target))
	if ext == ".cmd" || ext == ".bat" {
		return "", nil, fmt.Errorf("batch scripts (.cmd/.bat) are not supported; choose a native .exe or a Python, Node.js, or PowerShell script")
	}
	if runner == "" || strings.EqualFold(runner, "auto") {
		switch ext {
		case ".py":
			runner = "python"
		case ".js", ".mjs", ".cjs":
			runner = "node"
		case ".ps1":
			runner = "powershell"
		case "", ".exe":
			executable, err = native(target)
			return executable, nil, err
		default:
			return "", nil, fmt.Errorf("cannot automatically run %q; select an interpreter executable explicitly", ext)
		}
	}

	script, err := filepath.Abs(target)
	if err != nil {
		return "", nil, fmt.Errorf("script path: %w", err)
	}
	info, err := os.Stat(script)
	if err != nil {
		return "", nil, fmt.Errorf("script %q: %w", script, err)
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("script %q must be a regular file", script)
	}

	var args []string
	switch strings.ToLower(runner) {
	case "python":
		runner, args = "python", []string{"-u"}
	case "node":
		runner = "node"
	case "powershell":
		runner, args = "powershell", []string{"-NoProfile", "-File"}
	}
	executable, err = native(runner)
	if err != nil {
		return "", nil, fmt.Errorf("interpreter: %w", err)
	}
	switch strings.ToLower(filepath.Base(executable)) {
	case "powershell.exe", "pwsh.exe":
		// Explicit file mode prevents PowerShell from interpreting script argv
		// as a command string, including for custom interpreter paths.
		args = []string{"-NoProfile", "-File"}
	case "cmd.exe", "wsl.exe":
		return "", nil, fmt.Errorf("%q is a command-mode launcher and cannot be used as a script interpreter", executable)
	}
	return executable, append(args, script), nil
}

func native(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("cannot find executable %q: %w", name, err)
	}
	if !strings.EqualFold(filepath.Ext(path), ".exe") {
		return "", fmt.Errorf("%q is not a native .exe executable; command wrappers and shell command strings are not supported", name)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	// Windows App Execution Aliases (including python.exe) are reparse points,
	// not regular files. LookPath plus the .exe check also supports those aliases.
	if info.IsDir() {
		return "", fmt.Errorf("executable %q must not be a directory", path)
	}
	return path, nil
}
