package catalog

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

// Binary VDF parser for Steam files (shortcuts.vdf, config.vdf).
//
// Node layout: <type><key>\x00<value>
//   type 0x00 (map):    children until 0x08 terminator
//   type 0x01 (string): bytes until 0x00
//   type 0x02 (int32):  4 bytes little-endian (signed)
//
// shortcuts.vdf: root map "shortcuts" -> { "0": {...}, "1": {...}, ... }
// each shortcut is a map with capitalized string fields + signed int32 `appid`.

type nodeType byte

const (
	typeMap    nodeType = 0x00
	typeString nodeType = 0x01
	typeInt32  nodeType = 0x02
	typeEnd    byte     = 0x08
)

// Shortcut is one non-Steam shortcut entry.
type Shortcut struct {
	Index     int
	Name      string
	AppID     uint32 // canonical unsigned appid (signed & 0xFFFFFFFF)
	Exe       string
	StartDir  string
	LaunchOpt string
	Icon      string
	Tags      []string
	IsHidden  bool
}

// ParseShortcutsVDF parses a binary shortcuts.vdf file and returns sorted shortcuts.
func ParseShortcutsVDF(path string) ([]Shortcut, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	root, err := parseBinaryVDF(data)
	if err != nil {
		return nil, fmt.Errorf("parse shortcuts.vdf: %w", err)
	}
	return extractShortcuts(root)
}

func parseBinaryVDF(data []byte) (map[string]any, error) {
	key, val, _, err := parseNode(data, 0)
	if err != nil {
		return nil, err
	}
	if _, ok := val.(map[string]any); !ok {
		return nil, fmt.Errorf("root value for %q is not a map", key)
	}
	return map[string]any{key: val}, nil
}

func parseNode(data []byte, idx int) (key string, val any, next int, err error) {
	if idx >= len(data) {
		return "", nil, idx, fmt.Errorf("truncated node")
	}
	t := data[idx]
	idx++
	end := bytes.IndexByte(data[idx:], 0x00)
	if end < 0 {
		return "", nil, idx, fmt.Errorf("unterminated key at offset %d", idx)
	}
	key = string(data[idx : idx+end])
	idx += end + 1
	switch nodeType(t) {
	case typeMap:
		m := map[string]any{}
		for {
			if idx >= len(data) {
				return "", nil, idx, fmt.Errorf("unterminated map %q", key)
			}
			if data[idx] == typeEnd {
				return key, m, idx + 1, nil
			}
			ck, cv, next, err := parseNode(data, idx)
			if err != nil {
				return "", nil, next, err
			}
			m[ck] = cv
			idx = next
		}
	case typeString:
		end = bytes.IndexByte(data[idx:], 0x00)
		if end < 0 {
			return "", nil, idx, fmt.Errorf("unterminated string value for %q", key)
		}
		return key, string(data[idx : idx+end]), idx + end + 1, nil
	case typeInt32:
		if idx+4 > len(data) {
			return "", nil, idx, fmt.Errorf("truncated int32 for %q", key)
		}
		v := int32(binary.LittleEndian.Uint32(data[idx : idx+4]))
		return key, v, idx + 4, nil
	default:
		return "", nil, idx, fmt.Errorf("unknown node type %d for key %q", t, key)
	}
}

func extractShortcuts(root map[string]any) ([]Shortcut, error) {
	raw, ok := root["shortcuts"]
	if !ok {
		return nil, fmt.Errorf("no 'shortcuts' key")
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("'shortcuts' value is not a map")
	}
	out := make([]Shortcut, 0, len(m))
	for k, v := range m {
		sc, ok := v.(map[string]any)
		if !ok {
			continue
		}
		var s Shortcut
		fmt.Sscanf(k, "%d", &s.Index)
		s.Name = strField(sc, "AppName")
		if a, ok := sc["appid"].(int32); ok {
			s.AppID = uint32(a)
		}
		s.Exe = unquote(strField(sc, "Exe"))
		s.StartDir = unquote(strField(sc, "StartDir"))
		s.LaunchOpt = strField(sc, "LaunchOptions")
		s.Icon = strField(sc, "icon")
		if h, ok := sc["IsHidden"].(int32); ok && h != 0 {
			s.IsHidden = true
		}
		if tg, ok := sc["tags"].(map[string]any); ok {
			for _, tv := range tg {
				if st, ok := tv.(string); ok && st != "" {
					s.Tags = append(s.Tags, st)
				}
			}
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out, nil
}

func strField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
