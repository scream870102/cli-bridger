// Demo is a cooperative CLI used to exercise discovery and terminal rendering.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"cli-bridger/internal/protocol"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--cli-bridger-describe" {
		_ = json.NewEncoder(os.Stdout).Encode(descriptor())
		return
	}
	if len(os.Args) < 2 || os.Args[1] != "render" {
		fmt.Fprintln(os.Stderr, "Usage: demo render [--steps=20] [--delay=0.05] [--color] [--label=hello] [--source=path]")
		os.Exit(2)
	}
	f := flag.NewFlagSet("render", flag.ExitOnError)
	steps := f.Int("steps", 20, "progress steps")
	delay := f.Float64("delay", 0.05, "seconds between steps")
	color := f.Bool("color", false, "colored progress")
	label := f.String("label", "Hello CLI Bridger", "display label")
	source := f.String("source", "", "display a path (no file access)")
	_ = f.Parse(os.Args[2:])
	if *steps < 1 || *steps > 100 || *delay < 0 || *delay > 1 {
		fmt.Fprintln(os.Stderr, "steps must be 1..100 and delay 0..1")
		os.Exit(2)
	}
	fmt.Println(*label)
	fmt.Println("BRIDGER_DEMO_ROOT:", os.Getenv("BRIDGER_DEMO_ROOT"))
	if *source != "" {
		fmt.Println("Source:", *source)
	}
	for i := 0; i <= *steps; i++ {
		n := 30 * i / *steps
		prefix, suffix := "", ""
		if *color {
			prefix = "\x1b[36m"
			suffix = "\x1b[0m"
		}
		fmt.Printf("\r%s[%s%s] %3d%%%s", prefix, strings.Repeat("=", n), strings.Repeat(" ", 30-n), 100*i / *steps, suffix)
		time.Sleep(time.Duration(*delay * float64(time.Second)))
	}
	fmt.Println("\r\nDone. No files were modified.")
}

func descriptor() protocol.Descriptor {
	minSteps, maxSteps, minDelay, maxDelay := 1.0, 100.0, 0.0, 1.0
	minText, maxText := 1, 80
	return protocol.Descriptor{Version: "1", Name: "CLI Bridger Demo", Description: "A safe progress-bar demo. It displays parameters without modifying files.", Root: protocol.Command{ID: "root", Name: "", Description: "Choose render to demonstrate live terminal output.", Parameters: []protocol.Parameter{{ID: "demoRoot", Name: "Demo root directory", Description: "Override BRIDGER_DEMO_ROOT for this child process; the demo prints it without accessing files.", Env: "BRIDGER_DEMO_ROOT", Type: "path", PathKind: "directory", Default: "."}}, Commands: []protocol.Command{{ID: "render", Name: "render", Description: "Render a terminal progress bar", Parameters: []protocol.Parameter{
		{ID: "steps", Name: "Steps", Description: "Number of progress updates before completion; more steps take longer at the same delay.", Flag: "--steps", Type: "int", Required: true, Default: 20, Examples: []any{10, 50}, Limits: &protocol.Limits{Min: &minSteps, Max: &maxSteps}},
		{ID: "delay", Name: "Delay (seconds)", Description: "Seconds to pause between progress updates; use zero to finish immediately.", Flag: "--delay", Type: "float", Default: 0.05, Examples: []any{0.01, 0.1}, Limits: &protocol.Limits{Min: &minDelay, Max: &maxDelay}},
		{ID: "color", Name: "Colored output", Description: "Display the progress bar in cyan using ANSI terminal color codes.", Flag: "--color", Type: "bool", Default: true},
		{ID: "label", Name: "Progress label", Description: "Text printed above the progress bar; editable here when colored output is enabled.", Flag: "--label", Type: "string", Default: "Hello CLI Bridger", Examples: []any{"Building assets", "Processing files"}, Limits: &protocol.Limits{MinLength: &minText, MaxLength: &maxText}, DependsOn: &protocol.Dependency{ID: "color", Value: true}},
		{ID: "source", Name: "Source path", Flag: "--source", Type: "path", PathKind: "file", Description: "Shown in output; never opened or modified", Examples: []any{"example.txt"}},
	}}}}}
}
