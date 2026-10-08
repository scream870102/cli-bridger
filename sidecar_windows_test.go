package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

const sidecarJSON = `{"version":"1","name":"External CLI","description":"User supplied description","root":{"id":"root","description":"Run CLI","parameters":[{"id":"message","name":"Message","description":"Message to send","type":"string","flag":"--message"}]}}`

func writeSidecarFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestSidecarTargetsAndPreview(t *testing.T) {
	for _, extension := range []string{".exe", ".ps1"} {
		t.Run(extension, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "third party"+extension)
			// Neither this invalid executable nor this throwing script can supply discovery.
			writeSidecarFixture(t, target, "throw 'discovery must not execute'")
			file := target + ".cli-bridger.json"
			writeSidecarFixture(t, file, sidecarJSON)
			a := &App{}
			loaded, err := a.Describe(target, "auto")
			if err != nil {
				t.Fatal(err)
			}
			if loaded.DescriptionFile != file || loaded.Target != target || loaded.Raw != sidecarJSON {
				t.Fatalf("incorrect source metadata: %+v", loaded)
			}
			args, err := a.Preview(nil, map[string]any{"message": "hello world"}, map[string]bool{"message": true})
			if err != nil {
				t.Fatal(err)
			}
			want := []string{loaded.Executable}
			if extension == ".ps1" {
				want = append(want, "-NoProfile", "-File", target)
			}
			want = append(want, "--message=hello world")
			if !reflect.DeepEqual(args, want) {
				t.Fatalf("argv = %q, want %q", args, want)
			}
		})
	}
}

func TestSidecarFailuresDoNotDiscover(t *testing.T) {
	for _, kind := range []string{"invalid", "oversize", "directory", "unreadable"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "cli.ps1")
			marker := filepath.Join(dir, "executed")
			writeSidecarFixture(t, target, "Set-Content -LiteralPath '"+strings.ReplaceAll(marker, "'", "''")+"' -Value executed")
			file := target + ".cli-bridger.json"
			switch kind {
			case "invalid":
				writeSidecarFixture(t, file, `{}`)
			case "oversize":
				writeSidecarFixture(t, file, strings.Repeat(" ", 1024*1024+1))
			case "directory":
				if err := os.Mkdir(file, 0700); err != nil {
					t.Fatal(err)
				}
			case "unreadable":
				writeSidecarFixture(t, file, sidecarJSON)
				name, err := syscall.UTF16PtrFromString(file)
				if err != nil {
					t.Fatal(err)
				}
				handle, err := syscall.CreateFile(name, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
				if err != nil {
					t.Fatal(err)
				}
				defer syscall.CloseHandle(handle)
			}
			if _, err := (&App{}).Describe(target, "auto"); err == nil || !strings.Contains(err.Error(), "description file") {
				t.Fatalf("expected file-specific failure, got %v", err)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("discovery executed: %v", err)
			}
		})
	}
}

func TestSidecarReloadAndSourceTransitions(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "cli.ps1")
	marker := filepath.Join(dir, "executed")
	writeSidecarFixture(t, target, "Set-Content -LiteralPath '"+strings.ReplaceAll(marker, "'", "''")+"' -Value executed\n'"+sidecarJSON+"'")
	file := target + ".cli-bridger.json"
	a := &App{}
	loaded, err := a.Describe(target, "auto")
	if err != nil || loaded.DescriptionFile != "" {
		t.Fatalf("native fallback: %+v, %v", loaded, err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("native discovery did not run:", err)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	writeSidecarFixture(t, file, sidecarJSON)
	if _, err := a.Describe(target, "auto"); err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(sidecarJSON, "External CLI", "Updated CLI", 1)
	writeSidecarFixture(t, file, updated)
	loaded, err = a.Reload(nil, nil, nil)
	if err != nil || loaded.Descriptor.Name != "Updated CLI" || loaded.DescriptionFile != file {
		t.Fatalf("external reload: %+v, %v", loaded, err)
	}
	previous := a.descriptor
	writeSidecarFixture(t, file, `{}`)
	if _, err := a.Reload(nil, nil, nil); err == nil || a.descriptor != previous {
		t.Fatal("invalid reload must retain the previous descriptor")
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Reload(nil, nil, nil); err == nil || a.descriptor != previous || a.descriptionFile != file {
		t.Fatal("removed file must fail while retaining description and source")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("external load/reload invoked discovery: %v", err)
	}
	loaded, err = a.Describe(target, "auto")
	if err != nil || loaded.DescriptionFile != "" || a.descriptionFile != "" {
		t.Fatalf("explicit load must permit native source again: %+v, %v", loaded, err)
	}
}
