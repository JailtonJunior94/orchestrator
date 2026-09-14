package specs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var ErrNotJSONObject = errors.New("content is not a JSON object")

type jsonMember struct {
	key         string
	keyStart    int
	valueStart  int
	valueEnd    int
	commaBefore int
}

func (c *Catalog) scanJSONObject(raw []byte) ([]jsonMember, int, error) {
	i := c.skipJSONSpace(raw, 0)
	if i >= len(raw) || raw[i] != '{' {
		return nil, 0, ErrNotJSONObject
	}
	i++

	members := make([]jsonMember, 0, 8)
	pendingComma := -1
	for {
		i = c.skipJSONSpace(raw, i)
		if i >= len(raw) {
			return nil, 0, ErrNotJSONObject
		}
		if raw[i] == '}' {
			return members, i, nil
		}
		if raw[i] == ',' {
			pendingComma = i
			i++
			continue
		}
		keyStart := i
		keyEnd, key, ok := c.scanJSONString(raw, i)
		if !ok {
			return nil, 0, ErrNotJSONObject
		}
		i = c.skipJSONSpace(raw, keyEnd)
		if i >= len(raw) || raw[i] != ':' {
			return nil, 0, ErrNotJSONObject
		}
		valueStart := c.skipJSONSpace(raw, i+1)
		valueEnd, ok := c.scanJSONValue(raw, valueStart)
		if !ok {
			return nil, 0, ErrNotJSONObject
		}
		members = append(members, jsonMember{
			key:         key,
			keyStart:    keyStart,
			valueStart:  valueStart,
			valueEnd:    valueEnd,
			commaBefore: pendingComma,
		})
		pendingComma = -1
		i = valueEnd
	}
}

func (c *Catalog) skipJSONSpace(raw []byte, i int) int {
	for i < len(raw) {
		switch raw[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			return i
		}
	}
	return i
}

func (c *Catalog) scanJSONString(raw []byte, i int) (int, string, bool) {
	if i >= len(raw) || raw[i] != '"' {
		return 0, "", false
	}
	start := i
	i++
	for i < len(raw) {
		switch raw[i] {
		case '\\':
			i += 2
		case '"':
			var decoded string
			if err := json.Unmarshal(raw[start:i+1], &decoded); err != nil {
				return 0, "", false
			}
			return i + 1, decoded, true
		default:
			i++
		}
	}
	return 0, "", false
}

func (c *Catalog) scanJSONValue(raw []byte, i int) (int, bool) {
	if i >= len(raw) {
		return 0, false
	}
	switch raw[i] {
	case '"':
		end, _, ok := c.scanJSONString(raw, i)
		return end, ok
	case '{', '[':
		return c.scanJSONContainer(raw, i)
	default:
		return c.scanJSONScalar(raw, i)
	}
}

func (c *Catalog) scanJSONContainer(raw []byte, i int) (int, bool) {
	open := raw[i]
	closer := byte('}')
	if open == '[' {
		closer = ']'
	}
	depth := 0
	for i < len(raw) {
		switch raw[i] {
		case '"':
			end, _, ok := c.scanJSONString(raw, i)
			if !ok {
				return 0, false
			}
			i = end
			continue
		case open:
			depth++
		case closer:
			depth--
			if depth == 0 {
				return i + 1, true
			}
		}
		i++
	}
	return 0, false
}

func (c *Catalog) scanJSONScalar(raw []byte, i int) (int, bool) {
	start := i
	for i < len(raw) {
		switch raw[i] {
		case ',', '}', ']', ' ', '\t', '\n', '\r':
			return i, i > start
		default:
			i++
		}
	}
	return 0, false
}

func (c *Catalog) detectJSONIndent(raw []byte, members []jsonMember) string {
	if len(members) == 0 {
		return "  "
	}
	keyStart := members[0].keyStart
	lineStart := strings.LastIndexByte(string(raw[:keyStart]), '\n') + 1
	indent := string(raw[lineStart:keyStart])
	if strings.TrimLeft(indent, " \t") != "" || indent == "" {
		return "  "
	}
	return indent
}

func (c *Catalog) SetJSONTopLevelKey(raw []byte, key string, value any) ([]byte, error) {
	members, closeIndex, err := c.scanJSONObject(raw)
	if err != nil {
		return nil, err
	}

	indent := c.detectJSONIndent(raw, members)
	encoded, err := json.MarshalIndent(value, indent, indent)
	if err != nil {
		return nil, fmt.Errorf("encode value for key %q: %w", key, err)
	}
	encodedKey, err := json.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("encode key %q: %w", key, err)
	}

	for _, member := range members {
		if member.key != key {
			continue
		}
		out := make([]byte, 0, len(raw)+len(encoded))
		out = append(out, raw[:member.valueStart]...)
		out = append(out, encoded...)
		out = append(out, raw[member.valueEnd:]...)
		return out, nil
	}

	insertion := make([]byte, 0, len(encoded)+len(indent)+len(encodedKey)+4)
	insertAt := closeIndex
	if len(members) > 0 {
		insertAt = members[len(members)-1].valueEnd
		insertion = append(insertion, ',')
	}
	insertion = append(insertion, '\n')
	insertion = append(insertion, indent...)
	insertion = append(insertion, encodedKey...)
	insertion = append(insertion, ':', ' ')
	insertion = append(insertion, encoded...)
	if len(members) == 0 {
		insertion = append(insertion, '\n')
	}

	out := make([]byte, 0, len(raw)+len(insertion))
	out = append(out, raw[:insertAt]...)
	out = append(out, insertion...)
	out = append(out, raw[insertAt:]...)
	return out, nil
}

func (c *Catalog) DeleteJSONTopLevelKey(raw []byte, key string) ([]byte, error) {
	members, closeIndex, err := c.scanJSONObject(raw)
	if err != nil {
		return nil, err
	}

	for index, member := range members {
		if member.key != key {
			continue
		}
		start := member.keyStart
		end := member.valueEnd
		switch {
		case member.commaBefore >= 0:
			start = member.commaBefore
		case index+1 < len(members):
			end = members[index+1].keyStart
			if members[index+1].commaBefore >= 0 {
				end = members[index+1].commaBefore + 1
				end = c.skipJSONSpace(raw, end)
			}
		default:
			start = c.startOfEnclosingLine(raw, member.keyStart)
			end = closeIndex
		}
		out := make([]byte, 0, len(raw))
		out = append(out, raw[:start]...)
		out = append(out, raw[end:]...)
		return out, nil
	}
	return raw, nil
}

func (c *Catalog) startOfEnclosingLine(raw []byte, index int) int {
	lineStart := strings.LastIndexByte(string(raw[:index]), '\n')
	if lineStart == -1 {
		return index
	}
	return lineStart
}
