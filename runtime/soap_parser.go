package runtime

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

const SOAP11Namespace = "http://schemas.xmlsoap.org/soap/envelope/"
const SOAP12Namespace = "http://www.w3.org/2003/05/soap-envelope"

type ParsedG2SMessage struct {
	RootLocalName string
	RootNamespace string
	SOAPNamespace string
	HasEnvelope   bool
	HasBody       bool
	Fields        map[string]string
}

type ParsedG2SEnvelope struct {
	RootKind      string
	ClassName     string
	OperationName string
	HostID        string
	EGMID         string
	DateTimeSent  string
	SessionID     string
	RawRoot       string
	RootNamespace string
	SOAPNamespace string
	HasEnvelope   bool
	HasBody       bool
	Fields        map[string]string
}

func ParseG2SMessage(data []byte) (ParsedG2SMessage, error) {
	parsed, err := ParseG2SEnvelope(data)
	if err != nil {
		return ParsedG2SMessage{}, err
	}
	root := firstNonEmpty(parsed.OperationName, parsed.RawRoot)
	return ParsedG2SMessage{RootLocalName: root, RootNamespace: parsed.RootNamespace, SOAPNamespace: parsed.SOAPNamespace, HasEnvelope: parsed.HasEnvelope, HasBody: parsed.HasBody, Fields: parsed.Fields}, nil
}

func ParseG2SEnvelope(data []byte) (ParsedG2SEnvelope, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var stack []xml.StartElement
	var parsed ParsedG2SEnvelope
	parsed.Fields = map[string]string{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ParsedG2SEnvelope{}, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			parent := ""
			if len(stack) > 0 {
				parent = stack[len(stack)-1].Name.Local
			}
			stack = append(stack, t)
			local := t.Name.Local
			if parsed.RawRoot == "" {
				parsed.RawRoot = local
				parsed.RootNamespace = t.Name.Space
			}
			if strings.EqualFold(local, "Envelope") {
				parsed.HasEnvelope = true
				parsed.SOAPNamespace = t.Name.Space
				if parsed.SOAPNamespace != "" && parsed.SOAPNamespace != SOAP11Namespace && parsed.SOAPNamespace != SOAP12Namespace {
					return ParsedG2SEnvelope{}, fmt.Errorf("unsupported soap namespace %q", parsed.SOAPNamespace)
				}
				continue
			}
			if strings.EqualFold(local, "Body") && parsed.HasEnvelope {
				parsed.HasBody = true
				continue
			}
			if strings.EqualFold(local, "g2sMessage") {
				parsed.RootKind = "g2sMessage"
				parsed.RootNamespace = t.Name.Space
				continue
			}
			if strings.EqualFold(local, "g2sBody") {
				if parsed.RootKind == "" {
					parsed.RootKind = "g2sBody"
				}
				copyAttrs(parsed.Fields, t.Attr)
				parsed.HostID = firstNonEmpty(parsed.HostID, attrValue(t.Attr, "hostId"))
				parsed.EGMID = firstNonEmpty(parsed.EGMID, attrValue(t.Attr, "egmId"))
				parsed.DateTimeSent = firstNonEmpty(parsed.DateTimeSent, attrValue(t.Attr, "dateTimeSent"))
				continue
			}
			if strings.EqualFold(local, "g2sAck") {
				parsed.RootKind = "g2sAck"
				parsed.OperationName = "g2sAck"
				copyAttrs(parsed.Fields, t.Attr)
				parsed.HostID = firstNonEmpty(parsed.HostID, attrValue(t.Attr, "hostId"))
				parsed.EGMID = firstNonEmpty(parsed.EGMID, attrValue(t.Attr, "egmId"))
				parsed.DateTimeSent = firstNonEmpty(parsed.DateTimeSent, attrValue(t.Attr, "dateTimeSent"))
				continue
			}
			if isG2SClassContainer(parent) && parsed.ClassName == "" {
				parsed.ClassName = local
				copyAttrs(parsed.Fields, t.Attr)
				continue
			}
			if parsed.ClassName != "" && strings.EqualFold(parent, parsed.ClassName) && parsed.OperationName == "" {
				parsed.OperationName = local
				copyAttrs(parsed.Fields, t.Attr)
				parsed.SessionID = firstNonEmpty(parsed.SessionID, attrValue(t.Attr, "sessionId"))
				continue
			}
			if parsed.OperationName == "" && isLegacyOperationCandidate(local, parent) {
				parsed.RootKind = "legacy"
				parsed.OperationName = local
				parsed.RootNamespace = t.Name.Space
				copyAttrs(parsed.Fields, t.Attr)
				parsed.HostID = firstNonEmpty(parsed.HostID, attrValue(t.Attr, "hostId"))
				parsed.EGMID = firstNonEmpty(parsed.EGMID, attrValue(t.Attr, "egmId"))
				parsed.SessionID = firstNonEmpty(parsed.SessionID, attrValue(t.Attr, "sessionId"))
				continue
			}
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text == "" || len(stack) == 0 {
				continue
			}
			cur := stack[len(stack)-1]
			parsed.Fields[cur.Name.Local] = text
			switch cur.Name.Local {
			case "hostId", "hostID", "host_id":
				parsed.HostID = firstNonEmpty(parsed.HostID, text)
			case "egmId", "egmID", "egm_id":
				parsed.EGMID = firstNonEmpty(parsed.EGMID, text)
			case "sessionId", "session_id":
				parsed.SessionID = firstNonEmpty(parsed.SessionID, text)
			case "dateTimeSent":
				parsed.DateTimeSent = firstNonEmpty(parsed.DateTimeSent, text)
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if parsed.OperationName == "" {
		return ParsedG2SEnvelope{}, fmt.Errorf("g2s operation is required")
	}
	if parsed.RootKind == "" {
		parsed.RootKind = "legacy"
	}
	return parsed, nil
}

func isG2SClassContainer(parent string) bool {
	return strings.EqualFold(parent, "g2sBody")
}

func isLegacyOperationCandidate(local, parent string) bool {
	if strings.EqualFold(local, "Envelope") || strings.EqualFold(local, "Body") || strings.EqualFold(local, "g2sMessage") || strings.EqualFold(local, "g2sBody") {
		return false
	}
	return strings.EqualFold(parent, "Body") || parent == ""
}

func copyAttrs(fields map[string]string, attrs []xml.Attr) {
	for _, a := range attrs {
		fields[a.Name.Local] = a.Value
	}
}

func attrValue(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}
