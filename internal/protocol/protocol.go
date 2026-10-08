// Package protocol defines version 1 of the CLI Bridger discovery contract.
package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Descriptor struct {
	Version     string  `json:"version"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Root        Command `json:"root"`
}
type Command struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  []Parameter `json:"parameters,omitempty"`
	Commands    []Command   `json:"commands,omitempty"`
}
type Parameter struct {
	ID          string      `json:"id"`
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Flag        string      `json:"flag,omitempty"`
	Type        string      `json:"type"`
	Required    bool        `json:"required,omitempty"`
	Default     any         `json:"default,omitempty"`
	Examples    []any       `json:"examples,omitempty"`
	Limits      *Limits     `json:"limits,omitempty"`
	Enum        []any       `json:"enum,omitempty"`
	DependsOn   *Dependency `json:"dependsOn,omitempty"`
	PathKind    string      `json:"pathKind,omitempty"`
}
type Limits struct {
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	MinLength *int     `json:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
}
type Dependency struct {
	ID    string `json:"id"`
	Value any    `json:"value"`
}

var identifier = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]*$`)
var flagName = regexp.MustCompile(`^--?[A-Za-z][A-Za-z0-9-]*$`)

func Parse(data []byte) (*Descriptor, error) {
	var d Descriptor
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return nil, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("expected exactly one JSON object")
	}
	if err := d.validate(); err != nil {
		return nil, err
	}
	return &d, nil
}

func (d *Descriptor) validate() error {
	if d == nil || d.Version != "1" || strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("descriptor requires version 1 and a name")
	}
	seen := map[string]bool{}
	var visit func(Command, map[string]Parameter, bool) error
	visit = func(c Command, inherited map[string]Parameter, root bool) error {
		if !identifier.MatchString(c.ID) || seen[c.ID] {
			return fmt.Errorf("invalid or duplicate command id %q", c.ID)
		}
		seen[c.ID] = true
		if !root && !identifier.MatchString(c.Name) {
			return fmt.Errorf("invalid command name %q", c.Name)
		}
		scope := map[string]Parameter{}
		flags := map[string]bool{}
		for k, v := range inherited {
			scope[k] = v
			if v.Flag != "" {
				flags[v.Flag] = true
			}
		}
		optionalPositional := false
		for _, p := range c.Parameters {
			if !identifier.MatchString(p.ID) || seen[p.ID] {
				return fmt.Errorf("invalid or duplicate parameter id %q", p.ID)
			}
			seen[p.ID] = true
			switch p.Type {
			case "string", "int", "float", "path", "bool":
			default:
				return fmt.Errorf("%s: unsupported type", p.ID)
			}
			if p.Flag != "" {
				if !flagName.MatchString(p.Flag) || flags[p.Flag] {
					return fmt.Errorf("%s: invalid or duplicate flag", p.ID)
				}
				flags[p.Flag] = true
			} else {
				if p.Type == "bool" || len(c.Commands) > 0 || (p.Required && optionalPositional) {
					return fmt.Errorf("%s: ambiguous positional parameter", p.ID)
				}
				optionalPositional = optionalPositional || !p.Required
			}
			if p.PathKind != "" && (p.Type != "path" || (p.PathKind != "file" && p.PathKind != "directory" && p.PathKind != "save")) {
				return fmt.Errorf("%s: invalid pathKind", p.ID)
			}
			if l := p.Limits; l != nil {
				numeric := p.Type == "int" || p.Type == "float"
				textual := p.Type == "string" || p.Type == "path"
				if (!numeric && (l.Min != nil || l.Max != nil)) || (!textual && (l.MinLength != nil || l.MaxLength != nil)) || (l.Min != nil && l.Max != nil && *l.Min > *l.Max) || (l.MinLength != nil && *l.MinLength < 0) || (l.MaxLength != nil && *l.MaxLength < 0) || (l.MinLength != nil && l.MaxLength != nil && *l.MinLength > *l.MaxLength) {
					return fmt.Errorf("%s: invalid limits", p.ID)
				}
			}
			if p.DependsOn != nil {
				dep, ok := scope[p.DependsOn.ID]
				if !ok {
					return fmt.Errorf("%s: dependency must reference an earlier or ancestor parameter", p.ID)
				}
				if _, err := valueString(dep, p.DependsOn.Value); err != nil {
					return fmt.Errorf("%s: invalid dependency value: %w", p.ID, err)
				}
			}
			if p.Default != nil {
				if _, err := valueString(p, p.Default); err != nil {
					return fmt.Errorf("%s default: %w", p.ID, err)
				}
			}
			for _, v := range p.Enum {
				copy := p
				copy.Enum = nil
				if _, err := valueString(copy, v); err != nil {
					return fmt.Errorf("%s enum: %w", p.ID, err)
				}
			}
			for _, v := range p.Examples {
				if _, err := valueString(p, v); err != nil {
					return fmt.Errorf("%s example: %w", p.ID, err)
				}
			}
			scope[p.ID] = p
		}
		names := map[string]bool{}
		for _, child := range c.Commands {
			if names[child.Name] {
				return fmt.Errorf("duplicate command name %q", child.Name)
			}
			names[child.Name] = true
			if err := visit(child, scope, false); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(d.Root, map[string]Parameter{}, true)
}

func valueString(p Parameter, v any) (string, error) {
	var s string
	switch p.Type {
	case "string", "path":
		var ok bool
		s, ok = v.(string)
		if !ok || strings.ContainsRune(s, 0) {
			return "", fmt.Errorf("%s requires text without NUL", p.ID)
		}
		if p.Type == "path" && strings.TrimSpace(s) == "" {
			return "", fmt.Errorf("%s requires a path", p.ID)
		}
		if p.Flag == "" && strings.HasPrefix(s, "-") {
			return "", fmt.Errorf("%s positional cannot start with '-'", p.ID)
		}
		if l := p.Limits; l != nil {
			n := utf8.RuneCountInString(s)
			if (l.MinLength != nil && n < *l.MinLength) || (l.MaxLength != nil && n > *l.MaxLength) {
				return "", fmt.Errorf("%s length out of range", p.ID)
			}
		}
	case "bool":
		b, ok := v.(bool)
		if !ok {
			return "", fmt.Errorf("%s requires boolean", p.ID)
		}
		s = strconv.FormatBool(b)
	case "int", "float":
		var n float64
		switch x := v.(type) {
		case float64:
			n = x
		case float32:
			n = float64(x)
		case int:
			n = float64(x)
		case int64:
			n = float64(x)
		case json.Number:
			var err error
			n, err = x.Float64()
			if err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("%s requires number", p.ID)
		}
		if math.IsNaN(n) || math.IsInf(n, 0) || (p.Type == "int" && (math.Trunc(n) != n || math.Abs(n) > 9007199254740991)) {
			return "", fmt.Errorf("%s invalid number", p.ID)
		}
		if l := p.Limits; l != nil {
			if (l.Min != nil && n < *l.Min) || (l.Max != nil && n > *l.Max) {
				return "", fmt.Errorf("%s number out of range", p.ID)
			}
		}
		s = strconv.FormatFloat(n, 'f', -1, 64)
	}
	if len(p.Enum) > 0 {
		match := false
		copy := p
		copy.Enum = nil
		for _, item := range p.Enum {
			candidate, err := valueString(copy, item)
			if err == nil && candidate == s {
				match = true
				break
			}
		}
		if !match {
			return "", fmt.Errorf("%s value is not in enum", p.ID)
		}
	}
	return s, nil
}

// BuildArgs returns an argv slice, never a shell command. commandPath contains command IDs.
func BuildArgs(d *Descriptor, commandPath []string, values map[string]any, enabled map[string]bool) ([]string, error) {
	if err := d.validate(); err != nil {
		return nil, err
	}
	chain := []Command{d.Root}
	current := d.Root
	for _, id := range commandPath {
		found := false
		for _, c := range current.Commands {
			if c.ID == id {
				chain = append(chain, c)
				current = c
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown command %q", id)
		}
	}
	var args []string
	active := map[string]string{}
	params := map[string]Parameter{}
	for index, c := range chain {
		if index > 0 {
			args = append(args, c.Name)
		}
		missingPositional := false
		for _, p := range c.Parameters {
			params[p.ID] = p
			isActive := p.Required || enabled[p.ID]
			if dep := p.DependsOn; dep != nil {
				actual, ok := active[dep.ID]
				expected, _ := valueString(params[dep.ID], dep.Value)
				isActive = isActive && ok && actual == expected
			}
			if !isActive {
				if p.Flag == "" {
					missingPositional = true
				}
				continue
			}
			v, ok := values[p.ID]
			if !ok || v == nil {
				v = p.Default
			}
			if v == nil {
				return nil, fmt.Errorf("%s requires a value", p.ID)
			}
			s, err := valueString(p, v)
			if err != nil {
				return nil, err
			}
			active[p.ID] = s
			if p.Type == "bool" {
				if s == "true" {
					args = append(args, p.Flag)
				}
				continue
			}
			if p.Flag != "" {
				args = append(args, p.Flag+"="+s)
			} else {
				if missingPositional {
					return nil, fmt.Errorf("%s requires preceding positional parameters", p.ID)
				}
				args = append(args, s)
			}
		}
	}
	return args, nil
}
