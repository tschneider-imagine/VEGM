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

func ParseG2SMessage(data []byte) (ParsedG2SMessage, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var stack []xml.StartElement
	inBody := false
	var msg *xml.StartElement
	fields := map[string]string{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ParsedG2SMessage{}, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t)
			if len(stack) == 1 {
				if !strings.EqualFold(t.Name.Local, "Envelope") {
					return ParsedG2SMessage{}, fmt.Errorf("soap envelope is required")
				}
				if t.Name.Space != "" && t.Name.Space != SOAP11Namespace && t.Name.Space != SOAP12Namespace {
					return ParsedG2SMessage{}, fmt.Errorf("unsupported soap namespace %q", t.Name.Space)
				}
			}
			if len(stack) == 2 && strings.EqualFold(t.Name.Local, "Body") {
				inBody = true
				continue
			}
			if inBody && msg == nil {
				copy := t
				msg = &copy
				continue
			}
			if inBody && msg != nil && len(stack) == 4 {
				var value string
				if err := dec.DecodeElement(&value, &t); err != nil {
					return ParsedG2SMessage{}, err
				}
				fields[t.Name.Local] = strings.TrimSpace(value)
				stack = stack[:len(stack)-1]
			}
		case xml.EndElement:
			if len(stack) > 0 {
				if len(stack) == 2 && strings.EqualFold(stack[len(stack)-1].Name.Local, "Body") {
					inBody = false
				}
				stack = stack[:len(stack)-1]
			}
		}
	}
	if msg == nil {
		return ParsedG2SMessage{}, fmt.Errorf("soap body message is required")
	}
	soapNS := ""
	if len(stack) > 0 {
		soapNS = stack[0].Name.Space
	}
	return ParsedG2SMessage{RootLocalName: msg.Name.Local, RootNamespace: msg.Name.Space, SOAPNamespace: soapNS, HasEnvelope: true, HasBody: true, Fields: fields}, nil
}
