// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package codec decodes and encodes structured config files (JSON, YAML, TOML)
// through a generic map so that dirctl can upsert or remove only its own entry
// while leaving all sibling content intact.
package codec

import (
	"bytes"
	"encoding/json"
	"fmt"

	toml "github.com/pelletier/go-toml/v2"
	yaml "go.yaml.in/yaml/v3"
)

// Format identifies a supported config file encoding.
type Format int

const (
	// JSON is the encoding/json format.
	JSON Format = iota
	// YAML is the gopkg-compatible YAML format.
	YAML
	// TOML is the github.com/pelletier/go-toml/v2 format.
	TOML
)

// String returns the lowercase format name.
func (f Format) String() string {
	switch f {
	case JSON:
		return "json"
	case YAML:
		return "yaml"
	case TOML:
		return "toml"
	default:
		return "unknown"
	}
}

// Decode parses data into a generic string-keyed map. Empty input yields an
// empty (non-nil) map so callers can upsert into a fresh config.
func Decode(format Format, data []byte) (map[string]any, error) {
	m := map[string]any{}

	if len(bytes.TrimSpace(data)) == 0 {
		return m, nil
	}

	switch format {
	case JSON:
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber() // keep numbers stable instead of coercing ints to float64

		if err := dec.Decode(&m); err != nil {
			return nil, fmt.Errorf("decode json: %w", err)
		}
	case YAML:
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("decode yaml: %w", err)
		}
	case TOML:
		if err := toml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("decode toml: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format %d", format)
	}

	return m, nil
}

// Encode marshals a generic map back to bytes in the given format.
func Encode(format Format, m map[string]any) ([]byte, error) {
	switch format {
	case JSON:
		out, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode json: %w", err)
		}

		return append(out, '\n'), nil
	case YAML:
		out, err := yaml.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("encode yaml: %w", err)
		}

		return out, nil
	case TOML:
		out, err := toml.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("encode toml: %w", err)
		}

		return out, nil
	default:
		return nil, fmt.Errorf("unsupported format %d", format)
	}
}

// GetNested walks path through nested maps and returns the value at the leaf.
func GetNested(m map[string]any, path ...string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}

	current := m
	for i, key := range path {
		value, ok := current[key]
		if !ok {
			return nil, false
		}

		if i == len(path)-1 {
			return value, true
		}

		next, ok := asStringMap(value)
		if !ok {
			return nil, false
		}

		current = next
	}

	return nil, false
}

// SetNested sets value at path, creating intermediate maps as needed.
func SetNested(m map[string]any, value any, path ...string) {
	if len(path) == 0 {
		return
	}

	current := m
	for _, key := range path[:len(path)-1] {
		next, ok := asStringMap(current[key])
		if !ok {
			next = map[string]any{}
			current[key] = next
		}

		current = next
	}

	current[path[len(path)-1]] = value
}

// DeleteNested removes the leaf at path. It reports whether something was removed.
func DeleteNested(m map[string]any, path ...string) bool {
	if len(path) == 0 {
		return false
	}

	current := m
	for _, key := range path[:len(path)-1] {
		next, ok := asStringMap(current[key])
		if !ok {
			return false
		}

		current = next
	}

	leaf := path[len(path)-1]
	if _, ok := current[leaf]; !ok {
		return false
	}

	delete(current, leaf)

	return true
}

// asStringMap normalizes the two map shapes decoders may produce
// (map[string]any and map[any]any) into map[string]any.
func asStringMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case map[any]any:
		converted := make(map[string]any, len(typed))
		for k, v := range typed {
			ks, ok := k.(string)
			if !ok {
				return nil, false
			}

			converted[ks] = v
		}

		return converted, true
	default:
		return nil, false
	}
}
