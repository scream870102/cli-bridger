package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/UserExistsError/conpty"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestPTYHelper(t *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-2] != "--pty-helper" {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case "burst":
		for i := 0; i < 4096; i++ {
			fmt.Printf("TOKEN-%04d payload-abcdefghijklmnopqrstuv\r\n", i)
		}
		fmt.Print("FINAL-TAIL-COMPLETE\r\n")
		os.Exit(0)
	case "hold":
		fmt.Print("READY-TO-STOP\r\n")
		for {
			time.Sleep(time.Second)
		}
	}
	fmt.Printf("\x1b[32m%s\x1b[0m\r\n", os.Args[len(os.Args)-1])
	var input string
	fmt.Scanln(&input)
	fmt.Printf("\rprogress 100%% %s\r\n", input)
	os.Exit(0)
}

func TestConPTYFinalBurst(t *testing.T) {
	for attempt := 0; attempt < 3; attempt++ {
		p := startPTYHelper(t, "burst")
		var output bytes.Buffer
		done := make(chan struct{})
		go func() {
			defer close(done)
			buf := make([]byte, 8192) // Match the desktop reader's chunk size.
			for {
				n, err := p.Read(buf)
				output.Write(buf[:n])
				if err != nil {
					return
				}
				// Model small event-delivery delays before the next read.
				time.Sleep(time.Millisecond)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		code, err := p.Wait(ctx)
		cancel()
		_ = p.Close()
		awaitPTYReader(t, done)
		if err != nil || code != 0 {
			t.Fatalf("attempt %d exit=%d err=%v", attempt, code, err)
		}
		text := output.String()
		for i := 0; i < 4096; i++ {
			token := fmt.Sprintf("TOKEN-%04d", i)
			if count := strings.Count(text, token); count != 1 {
				t.Fatalf("attempt %d: %s count=%d, captured=%d bytes, tail=%t", attempt, token, count, len(text), strings.Contains(text, "FINAL-TAIL-COMPLETE"))
			}
		}
		if !strings.Contains(text, "FINAL-TAIL-COMPLETE") {
			t.Fatalf("attempt %d missing final tail", attempt)
		}
	}
}

func TestConPTYStopAndFreshSession(t *testing.T) {
	p := startPTYHelper(t, "hold")
	process, err := os.FindProcess(p.Pid())
	if err != nil {
		t.Fatal(err)
	}
	defer process.Release()
	done := make(chan struct{})
	ready := make(chan struct{})
	go func() {
		defer close(done)
		var initial strings.Builder
		buf := make([]byte, 8192)
		signaled := false
		for {
			n, err := p.Read(buf)
			if !signaled {
				initial.Write(buf[:n])
				if strings.Contains(initial.String(), "READY-TO-STOP") {
					close(ready)
					signaled = true
				}
			}
			if err != nil {
				return
			}
		}
	}()
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		_ = p.Close()
		t.Fatal("helper did not become ready")
	}
	waitDone := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() { _, err := p.Wait(ctx); waitDone <- err }()
	if err := p.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
	awaitPTYReader(t, done)
	select {
	case err := <-waitDone:
		t.Logf("wait after stop: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("wait did not finish after stop")
	}
	terminated := make(chan error, 1)
	go func() { _, err := process.Wait(); terminated <- err }()
	select {
	case err := <-terminated:
		if err != nil {
			t.Fatalf("process termination: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("child process survived stop")
	}
	// A fresh interactive session must still accept input and finish normally.
	t.Run("fresh", TestConPTYRoundTrip)
}

func startPTYHelper(t *testing.T, mode string) *conpty.ConPty {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	args := []string{executable, "-test.run=^TestPTYHelper$", "--", "--pty-helper", mode}
	for i := range args {
		args[i] = syscall.EscapeArg(args[i])
	}
	p, err := conpty.Start(strings.Join(args, " "), conpty.ConPtyDimensions(110, 30))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func awaitPTYReader(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("PTY reader did not stop")
	}
}

func TestConPTYRoundTrip(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	label := `中文 path with spaces & "quotes"`
	args := []string{executable, "-test.run=^TestPTYHelper$", "--", "--pty-helper", label}
	for i := range args {
		args[i] = syscall.EscapeArg(args[i])
	}
	p, err := conpty.Start(strings.Join(args, " "), conpty.ConPtyDimensions(90, 24))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	done := make(chan struct{})
	go func() { _, _ = io.Copy(&output, p); close(done) }()
	if err = p.Resize(110, 30); err != nil {
		t.Error(err)
	}
	if _, err = p.Write([]byte("accepted\r\n")); err != nil {
		t.Error(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	code, waitErr := p.Wait(ctx)
	_ = p.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("PTY reader did not stop")
	}
	if waitErr != nil || code != 0 {
		t.Fatalf("exit=%d err=%v output=%q", code, waitErr, output.String())
	}
	for _, want := range []string{label, "100% accepted", "\x1b[", "\r"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("missing %q in %q", want, output.String())
		}
	}
}
