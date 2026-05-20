package runtime

import (
	"bytes"
	"encoding/xml"
	"regexp"
	"strings"
)

type ParsedMessage struct {
	RootLocalName string
	Fields        map[string]string
}

func ParseMessage(raw []byte) ParsedMessage {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	fields := map[string]string{}
	stack := []xml.StartElement{}
	var root string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t)
			if root == "" && !isEnvelopeLocal(t.Name.Local) && !isBodyLocal(t.Name.Local) {
				root = t.Name.Local
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text == "" || len(stack) == 0 {
				continue
			}
			cur := stack[len(stack)-1]
			fields[cur.Name.Local] = text
		}
	}
	if root == "" {
		if m := rootLocalNameFromRaw(raw); m != "" {
			root = m
		}
	}
	return ParsedMessage{RootLocalName: root, Fields: fields}
}

func rootLocalNameFromRaw(raw []byte) string {
	s := string(raw)
	re := regexp.MustCompile(`(?s)<[^>]*Body[^>]*>\s*<(?:(?:[A-Za-z0-9_\-]+):)?([A-Za-z0-9_\-]+)`)
	if m := re.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	re2 := regexp.MustCompile(`(?s)<(?:(?:[A-Za-z0-9_\-]+):)?([A-Za-z0-9_\-]+)[^>]*>`)
	if m := re2.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	return ""
}

func isEnvelopeLocal(local string) bool {
	return strings.EqualFold(local, "Envelope")
}

func isBodyLocal(local string) bool {
	return strings.EqualFold(local, "Body")
}

func ExtractLocalName(selector string) string {
	re := regexp.MustCompile(`local-name\(\)='([^']+)'`)
	if m := re.FindStringSubmatch(selector); len(m) == 2 {
		return m[1]
	}
	return ""
}
