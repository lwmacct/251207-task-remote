package appcmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

func utilReplaceJSONStringValue(content []byte, path []string, value string) ([]byte, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("empty JSON path")
	}

	cursor := jsonEditCursor{content: content}
	start, end, err := cursor.findStringValue(path)
	if err != nil {
		return nil, err
	}

	quoted, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	next := make([]byte, 0, len(content)-(end-start)+len(quoted))
	next = append(next, content[:start]...)
	next = append(next, quoted...)
	next = append(next, content[end:]...)
	return next, nil
}

type jsonEditCursor struct {
	content []byte
	offset  int
}

func (c *jsonEditCursor) findStringValue(path []string) (int, int, error) {
	c.skipWhitespace()
	start, end, err := c.findStringValueInObject(path)
	if err != nil {
		return 0, 0, err
	}
	c.skipWhitespace()
	return start, end, nil
}

func (c *jsonEditCursor) findStringValueInObject(path []string) (int, int, error) {
	if err := c.expectByte('{'); err != nil {
		return 0, 0, err
	}
	c.skipWhitespace()
	if c.consumeByte('}') {
		return 0, 0, fmt.Errorf("json path not found: %s", jsonPathName(path))
	}

	for {
		key, _, _, err := c.parseString()
		if err != nil {
			return 0, 0, err
		}
		c.skipWhitespace()
		if err := c.expectByte(':'); err != nil {
			return 0, 0, err
		}
		c.skipWhitespace()

		if key == path[0] {
			if len(path) == 1 {
				_, start, end, err := c.parseString()
				if err != nil {
					return 0, 0, fmt.Errorf("json path is not a string value: %s", jsonPathName(path))
				}
				return start, end, nil
			}
			return c.findStringValueInObject(path[1:])
		}

		if err := c.skipValue(); err != nil {
			return 0, 0, err
		}
		c.skipWhitespace()
		if c.consumeByte('}') {
			return 0, 0, fmt.Errorf("json path not found: %s", jsonPathName(path))
		}
		if err := c.expectByte(','); err != nil {
			return 0, 0, err
		}
		c.skipWhitespace()
	}
}

func (c *jsonEditCursor) skipValue() error {
	c.skipWhitespace()
	if c.offset >= len(c.content) {
		return fmt.Errorf("unexpected end of JSON")
	}

	switch c.content[c.offset] {
	case '"':
		_, _, _, err := c.parseString()
		return err
	case '{':
		return c.skipObject()
	case '[':
		return c.skipArray()
	default:
		start := c.offset
		for c.offset < len(c.content) && !isJSONDelimiter(c.content[c.offset]) {
			c.offset++
		}
		if start == c.offset {
			return fmt.Errorf("unexpected JSON token at byte %d", c.offset)
		}
		return nil
	}
}

func (c *jsonEditCursor) skipObject() error {
	if err := c.expectByte('{'); err != nil {
		return err
	}
	c.skipWhitespace()
	if c.consumeByte('}') {
		return nil
	}

	for {
		if _, _, _, err := c.parseString(); err != nil {
			return err
		}
		c.skipWhitespace()
		if err := c.expectByte(':'); err != nil {
			return err
		}
		if err := c.skipValue(); err != nil {
			return err
		}
		c.skipWhitespace()
		if c.consumeByte('}') {
			return nil
		}
		if err := c.expectByte(','); err != nil {
			return err
		}
		c.skipWhitespace()
	}
}

func (c *jsonEditCursor) skipArray() error {
	if err := c.expectByte('['); err != nil {
		return err
	}
	c.skipWhitespace()
	if c.consumeByte(']') {
		return nil
	}

	for {
		if err := c.skipValue(); err != nil {
			return err
		}
		c.skipWhitespace()
		if c.consumeByte(']') {
			return nil
		}
		if err := c.expectByte(','); err != nil {
			return err
		}
		c.skipWhitespace()
	}
}

func (c *jsonEditCursor) parseString() (string, int, int, error) {
	start := c.offset
	if err := c.expectByte('"'); err != nil {
		return "", 0, 0, err
	}

	escaped := false
	for c.offset < len(c.content) {
		ch := c.content[c.offset]
		c.offset++
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			end := c.offset
			var value string
			if err := json.Unmarshal(c.content[start:end], &value); err != nil {
				return "", 0, 0, err
			}
			return value, start, end, nil
		}
	}
	return "", 0, 0, fmt.Errorf("unterminated JSON string at byte %d", start)
}

func (c *jsonEditCursor) expectByte(expected byte) error {
	if c.offset >= len(c.content) {
		return fmt.Errorf("expected %q at end of JSON", expected)
	}
	if c.content[c.offset] != expected {
		return fmt.Errorf("expected %q at byte %d", expected, c.offset)
	}
	c.offset++
	return nil
}

func (c *jsonEditCursor) consumeByte(expected byte) bool {
	if c.offset < len(c.content) && c.content[c.offset] == expected {
		c.offset++
		return true
	}
	return false
}

func (c *jsonEditCursor) skipWhitespace() {
	for c.offset < len(c.content) && unicode.IsSpace(rune(c.content[c.offset])) {
		c.offset++
	}
}

func isJSONDelimiter(ch byte) bool {
	return ch == ',' || ch == '}' || ch == ']' || unicode.IsSpace(rune(ch))
}

func jsonPathName(path []string) string {
	parts := make([]string, len(path))
	for i, item := range path {
		parts[i] = fmt.Sprintf("[%q]", item)
	}
	return strings.Join(parts, "")
}
