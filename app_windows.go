package main

import (
	"bytes"
	"cli-bridger/internal/launcher"
	"cli-bridger/internal/protocol"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/UserExistsError/conpty"
	"github.com/wailsapp/wails/v3/pkg/application"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

type App struct {
	desktop    *application.App
	mu         sync.Mutex
	executable string
	prefix     []string
	descriptor *protocol.Descriptor
	terminal   *conpty.ConPty
}
type Loaded struct {
	Raw        string               `json:"raw"`
	Descriptor *protocol.Descriptor `json:"descriptor"`
	Executable string               `json:"executable"`
	Target     string               `json:"target"`
}

// A faulty CLI cannot fill memory indefinitely during discovery.
type boundedBuffer struct{ bytes.Buffer }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 1024*1024 {
		return 0, errors.New("description exceeds 1 MiB")
	}
	return b.Buffer.Write(p)
}
func (a *App) Describe(target, runner string) (*Loaded, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.terminal != nil {
		return nil, errors.New("stop the current command first")
	}
	a.descriptor = nil
	a.executable = ""
	a.prefix = nil
	path, prefix, err := launcher.Resolve(target, runner)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, append(append([]string{}, prefix...), "--cli-bridger-describe")...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.WaitDelay = time.Second
	var out, diagnostic boundedBuffer
	cmd.Stdout = &out
	cmd.Stderr = &diagnostic
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("CLI description failed (requires --cli-bridger-describe): %w; %s", err, diagnostic.String())
	}
	d, err := protocol.Parse(out.Bytes())
	if err != nil {
		return nil, err
	}
	a.descriptor = d
	a.executable = path
	a.prefix = prefix
	resolvedTarget := path
	if len(prefix) > 0 {
		resolvedTarget = prefix[len(prefix)-1]
	}
	return &Loaded{Raw: out.String(), Descriptor: d, Executable: path, Target: resolvedTarget}, nil
}
func (a *App) PickPath(kind string) (string, error) {
	switch kind {
	case "directory":
		return a.desktop.Dialog.OpenFile().CanChooseFiles(false).CanChooseDirectories(true).SetTitle("選擇資料夾").PromptForSingleSelection()
	case "save":
		return a.desktop.Dialog.SaveFile().SetMessage("選擇輸出路徑").PromptForSingleSelection()
	default:
		return a.desktop.Dialog.OpenFile().SetTitle("選擇檔案").PromptForSingleSelection()
	}
}
func (a *App) Preview(path []string, values map[string]any, enabled map[string]bool) ([]string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.descriptor == nil {
		return nil, errors.New("load a CLI description first")
	}
	args, err := protocol.BuildArgs(a.descriptor, path, values, enabled)
	return append(append([]string{a.executable}, a.prefix...), args...), err
}
func (a *App) Run(path []string, values map[string]any, enabled map[string]bool, cols, rows int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.terminal != nil {
		return errors.New("a command is already running")
	}
	if a.descriptor == nil {
		return errors.New("load a CLI description first")
	}
	args, err := protocol.BuildArgs(a.descriptor, path, values, enabled)
	if err != nil {
		return err
	}
	parts := append(append([]string{a.executable}, a.prefix...), args...)
	for i := range parts {
		parts[i] = syscall.EscapeArg(parts[i])
	}
	cols, rows = dimensions(cols, rows)
	p, err := conpty.Start(strings.Join(parts, " "), conpty.ConPtyDimensions(cols, rows))
	if err != nil {
		return err
	}
	a.terminal = p
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 8192)
		for {
			n, e := p.Read(buf)
			if n > 0 {
				a.desktop.Event.Emit("terminal:data", base64.StdEncoding.EncodeToString(buf[:n]))
			}
			if e != nil {
				return
			}
		}
	}()
	go func() {
		code, e := p.Wait(context.Background())
		// Closing the pseudoconsole flushes output; reader keeps draining during Close.
		a.mu.Lock()
		stopped := a.terminal != p
		if a.terminal == p {
			_ = p.Close()
			a.terminal = nil
		}
		a.mu.Unlock()
		<-done
		message := fmt.Sprintf("Process exited with code %d", code)
		if stopped {
			message = "Process stopped"
		} else if e != nil {
			message = e.Error()
		}
		a.desktop.Event.Emit("terminal:exit", message)
	}()
	return nil
}
func dimensions(cols, rows int) (int, int) { return max(2, min(cols, 500)), max(2, min(rows, 200)) }
func (a *App) Input(data string) error {
	a.mu.Lock()
	p := a.terminal
	a.mu.Unlock()
	if p == nil {
		return nil
	}
	// A blocked stdin pipe must not prevent Stop from closing the pseudoconsole.
	_, err := p.Write([]byte(data))
	return err
}
func (a *App) Resize(cols, rows int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.terminal == nil {
		return nil
	}
	cols, rows = dimensions(cols, rows)
	return a.terminal.Resize(cols, rows)
}
func (a *App) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.terminal == nil {
		return nil
	}
	err := a.terminal.Close()
	a.terminal = nil
	return err
}
