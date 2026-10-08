package protocol

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const sample = `{"version":"1","name":"demo","root":{"id":"root","name":"","parameters":[{"id":"verbose","flag":"--verbose","type":"bool","default":false}],"commands":[{"id":"run","name":"run","parameters":[{"id":"count","flag":"--count","type":"int","required":true,"default":3,"limits":{"min":1,"max":5}},{"id":"mode","flag":"--mode","type":"string","default":"fast","enum":["fast","slow"]},{"id":"detail","flag":"--detail","type":"string","required":true,"dependsOn":{"id":"verbose","value":true}},{"id":"input","type":"path","required":true}]}]}}`

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
