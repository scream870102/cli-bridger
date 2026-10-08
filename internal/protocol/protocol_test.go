package protocol

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const sample = `{"version":"1","name":"demo","description":"Demonstrate typed arguments","root":{"id":"root","name":"","description":"Choose an operation","parameters":[{"id":"verbose","description":"Show detailed output","flag":"--verbose","type":"bool","default":false}],"commands":[{"id":"run","name":"run","description":"Process an input file","parameters":[{"id":"count","description":"Number of processing iterations","flag":"--count","type":"int","required":true,"default":3,"limits":{"min":1,"max":5}},{"id":"mode","description":"Select processing speed","flag":"--mode","type":"string","default":"fast","enum":["fast","slow"]},{"id":"detail","description":"Text to include in verbose output","flag":"--detail","type":"string","required":true,"dependsOn":{"id":"verbose","value":true}},{"id":"input","description":"Path of the input file","type":"path","required":true}]}]}}`

func TestBuildArgs(t *testing.T) {
	d, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		values  map[string]any
		enabled map[string]bool
		want    []string
		bad     string
	}{
		{"defaults", map[string]any{"input": "a b.txt"}, nil, []string{"run", "--count=3", "a b.txt"}, ""},
		{"dependency inactive", map[string]any{"input": "file", "detail": "ignored"}, nil, []string{"run", "--count=3", "file"}, ""},
		{"required dependency", map[string]any{"input": "file", "verbose": true}, map[string]bool{"verbose": true}, nil, "detail"},
		{"dependency active", map[string]any{"input": "file", "verbose": true, "detail": "a; echo bad"}, map[string]bool{"verbose": true}, []string{"--verbose", "run", "--count=3", "--detail=a; echo bad", "file"}, ""},
		{"range", map[string]any{"input": "file", "count": 6}, nil, nil, "range"},
		{"fraction", map[string]any{"input": "file", "count": 1.5}, nil, nil, "number"},
		{"numeric string", map[string]any{"input": "file", "count": "2"}, nil, nil, "number"},
		{"missing required", nil, nil, nil, "input"},
		{"positional injection", map[string]any{"input": "--delete"}, nil, nil, "positional"},
		{"enum", map[string]any{"input": "file", "mode": "other"}, map[string]bool{"mode": true}, nil, "enum"},
		{"optional explicit", map[string]any{"input": "file", "mode": "slow"}, nil, []string{"run", "--count=3", "file"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildArgs(d, []string{"run"}, tt.values, tt.enabled)
			if tt.bad != "" {
				if err == nil || !strings.Contains(err.Error(), tt.bad) {
					t.Fatalf("expected %s, got %v", tt.bad, err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
	if _, err := BuildArgs(d, []string{"unknown"}, nil, nil); err == nil {
		t.Fatal("unknown command accepted")
	}
}

func TestRejectInvalidSchema(t *testing.T) {
	if _, err := Parse([]byte(sample)); err != nil {
		t.Fatalf("invalid baseline fixture: %v", err)
	}
	for _, pair := range [][2]string{{`"version":"1"`, `"version":"2"`}, {`"id":"count"`, `"id":"verbose"`}, {`"type":"int"`, `"type":"integer"`}, {`"default":3`, `"default":7`}, {`"min":1`, `"min":9`}, {`"id":"verbose","value":true`, `"id":"missing","value":true`}, {`"type":"path"`, `"type":"path","pathKind":"device"`}, {`"flag":"--count"`, `"flag":"--count;rm"`}} {
		if _, err := Parse([]byte(strings.Replace(sample, pair[0], pair[1], 1))); err == nil {
			t.Errorf("accepted replacement %s", pair[1])
		}
	}
	if _, err := Parse([]byte(sample + ` {}`)); err == nil {
		t.Fatal("trailing JSON accepted")
	}
	var d Descriptor
	if err := json.Unmarshal([]byte(sample), &d); err != nil {
		t.Fatal(err)
	}
	d.Root.Parameters[0].Limits = &Limits{Min: new(float64)}
	if _, err := BuildArgs(&d, nil, nil, nil); err == nil {
		t.Fatal("invalid direct descriptor accepted")
	}
}

func TestDescriptionsRequired(t *testing.T) {
	for _, target := range []struct{ description, context string }{
		{"Demonstrate typed arguments", `tool "demo"`},
		{"Choose an operation", `command "root"`},
		{"Process an input file", `command "run"`},
		{"Show detailed output", `parameter "verbose"`},
		{"Number of processing iterations", `parameter "count"`},
	} {
		for _, replacement := range []string{"", `"description":"",`, `"description":" \t\n\u3000",`} {
			t.Run(target.context+"/"+replacement, func(t *testing.T) {
				data := strings.Replace(sample, `"description":"`+target.description+`",`, replacement, 1)
				_, err := Parse([]byte(data))
				if err == nil || !strings.Contains(err.Error(), target.context+": description") {
					t.Fatalf("expected description error identifying %s, got %v", target.context, err)
				}
			})
		}
	}
}

func TestEnvironmentBindings(t *testing.T) {
	d, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	d.Root.Parameters = append(d.Root.Parameters,
		Parameter{ID: "media", Description: "Media directory", Env: "ZZZ_MEDIA_ROOT", Type: "path", PathKind: "directory", Default: "."},
		Parameter{ID: "debug", Description: "Debug environment", Env: "APP_DEBUG", Type: "bool", Default: false, DependsOn: &Dependency{ID: "verbose", Value: true}},
	)
	tests := []struct {
		name    string
		values  map[string]any
		enabled map[string]bool
		want    map[string]string
		bad     string
	}{
		{"disabled inherits", nil, nil, map[string]string{}, ""},
		{"enabled default", nil, map[string]bool{"media": true}, map[string]string{"ZZZ_MEDIA_ROOT": "."}, ""},
		{"literal env text", map[string]any{"media": "-path with spaces;literal"}, map[string]bool{"media": true}, map[string]string{"ZZZ_MEDIA_ROOT": "-path with spaces;literal"}, ""},
		{"bad path", map[string]any{"media": " "}, map[string]bool{"media": true}, nil, "media"},
		{"NUL", map[string]any{"media": "a\x00b"}, map[string]bool{"media": true}, nil, "NUL"},
		{"inactive dependency", nil, map[string]bool{"debug": true}, map[string]string{}, ""},
		{"false bool", map[string]any{"verbose": true}, map[string]bool{"verbose": true, "debug": true}, map[string]string{"APP_DEBUG": "false"}, ""},
		{"true bool", map[string]any{"verbose": true, "debug": true}, map[string]bool{"verbose": true, "debug": true}, map[string]string{"APP_DEBUG": "true"}, ""},
		{"invalid parent", map[string]any{"verbose": "true"}, map[string]bool{"verbose": true, "debug": true}, nil, "verbose"},
		{"disabled env ignores parent", map[string]any{"verbose": "invalid"}, map[string]bool{"verbose": true}, map[string]string{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The required positional input is intentionally absent on every reload.
			got, err := BuildEnvironment(d, []string{"run"}, tt.values, tt.enabled)
			if tt.bad != "" {
				if err == nil || !strings.Contains(err.Error(), tt.bad) {
					t.Fatalf("wanted %s, got %v", tt.bad, err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v %v; want %#v", got, err, tt.want)
			}
		})
	}
	args, err := BuildArgs(d, []string{"run"}, map[string]any{"input": "file", "media": "root"}, map[string]bool{"media": true})
	if err != nil || !reflect.DeepEqual(args, []string{"run", "--count=3", "file"}) {
		t.Fatalf("env leaked into argv: %q %v", args, err)
	}
	if _, err := BuildArgs(d, []string{"run"}, map[string]any{"input": "file", "media": ""}, map[string]bool{"media": true}); err == nil {
		t.Fatal("argv builder did not validate env")
	}
	d.Root.Parameters[1].Required = true
	d.Root.Parameters[1].Default = nil
	if _, err := BuildEnvironment(d, []string{"run"}, nil, nil); err == nil || !strings.Contains(err.Error(), "media") {
		t.Fatalf("missing required env accepted: %v", err)
	}
	got, err := BuildEnvironment(d, []string{"run"}, map[string]any{"media": "."}, nil)
	if err != nil || got["ZZZ_MEDIA_ROOT"] != "." {
		t.Fatalf("required env not enabled: %v %v", got, err)
	}
}

func TestInvalidEnvironmentSchema(t *testing.T) {
	for _, name := range []string{"1ROOT", "ROOT-NAME", "ROOT=BAD", "ROOT\x00BAD"} {
		d, _ := Parse([]byte(sample))
		d.Root.Parameters = append(d.Root.Parameters, Parameter{ID: "env", Description: "Environment", Env: name, Type: "string"})
		data, _ := json.Marshal(d)
		if _, err := Parse(data); err == nil {
			t.Errorf("accepted invalid env name %q", name)
		}
	}
	d, _ := Parse([]byte(sample))
	d.Root.Parameters[0].Env = "APP_VERBOSE"
	data, _ := json.Marshal(d)
	if _, err := Parse(data); err == nil {
		t.Fatal("accepted flag and env together")
	}
	d.Root.Parameters[0].Flag = ""
	d.Root.Commands[0].Parameters = append(d.Root.Commands[0].Parameters, Parameter{ID: "duplicate", Description: "Duplicate inherited binding", Env: "app_verbose", Type: "bool"})
	data, _ = json.Marshal(d)
	if _, err := Parse(data); err == nil {
		t.Fatal("accepted inherited case-insensitive duplicate env")
	}
}

func TestCustomEnvironment(t *testing.T) {
	d, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	d.Root.Commands[0].Parameters = append(d.Root.Commands[0].Parameters, Parameter{ID: "media", Description: "Media directory", Env: "APP_MEDIA", Type: "path"})
	got, err := CustomEnvironment(d, []string{"run"}, []EnvironmentVariable{{"HTTP_PROXY", "http://127.0.0.1:7890"}, {"ProgramFiles(x86)", "中文=a"}, {"EMPTY", ""}})
	want := map[string]string{"HTTP_PROXY": "http://127.0.0.1:7890", "ProgramFiles(x86)": "中文=a", "EMPTY": ""}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v %v", got, err)
	}
	// A subcommand's declaration only owns the name when that subcommand is selected.
	if got, err := CustomEnvironment(d, nil, []EnvironmentVariable{{"app_media", "x"}}); err != nil || got["app_media"] != "x" {
		t.Fatalf("root path rejected unrelated name: %#v %v", got, err)
	}
	for _, tt := range []struct {
		name   string
		custom []EnvironmentVariable
		bad    string
	}{
		{"empty name", []EnvironmentVariable{{"", "x"}}, "invalid"},
		{"padded name", []EnvironmentVariable{{" A", "x"}}, "invalid"},
		{"equals", []EnvironmentVariable{{"A=B", "x"}}, "invalid"},
		{"NUL name", []EnvironmentVariable{{"A\x00", "x"}}, "invalid"},
		{"NUL value", []EnvironmentVariable{{"A", "a\x00b"}}, "NUL"},
		{"declared", []EnvironmentVariable{{"app_media", "x"}}, "provided by the tool"},
		{"duplicate", []EnvironmentVariable{{"Proxy", "a"}, {"PROXY", "b"}}, "duplicate"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := CustomEnvironment(d, []string{"run"}, tt.custom); err == nil || !strings.Contains(err.Error(), tt.bad) {
				t.Fatalf("wanted %s, got %v", tt.bad, err)
			}
		})
	}
	if _, err := CustomEnvironment(d, []string{"missing"}, nil); err == nil {
		t.Fatal("unknown command accepted")
	}
}
