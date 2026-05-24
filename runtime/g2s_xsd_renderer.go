package runtime

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"time"
)

const G2SXMLModeLegacy = "lab_legacy_xml"
const G2SXMLModeXSDMessage = "xsd_g2s_message"
const G2SDefaultNamespace = "http://www.gamingstandards.com/g2s/schemas/v1.0.3"

func (s *Server) shouldRenderXSDG2SMessage() bool {
	return s != nil && s.cfg != nil && s.cfg.G2SXML.Mode == G2SXMLModeXSDMessage
}

func (s *Server) renderXSDCommunicationsMessage(operationName string, attrs map[string]string) string {
	if attrs == nil {
		attrs = map[string]string{}
	}
	ns := firstNonEmpty(s.cfg.G2SXML.Namespace, G2SDefaultNamespace)
	bodyAttrs := map[string]string{
		"hostId":       s.cfg.HostID,
		"egmId":        s.cfg.EGMID,
		"dateTimeSent": time.Now().UTC().Format(time.RFC3339Nano),
	}
	var b bytes.Buffer
	b.WriteString(`<g2s:g2sMessage xmlns:g2s="`)
	b.WriteString(xmlEscape(ns))
	b.WriteString(`"><g2s:g2sBody`)
	writeXMLAttrs(&b, bodyAttrs)
	b.WriteString(`><g2s:communications><g2s:`)
	b.WriteString(operationName)
	writeXMLAttrs(&b, attrs)
	b.WriteString(` /></g2s:communications></g2s:g2sBody></g2s:g2sMessage>`)
	return b.String()
}

func writeXMLAttrs(b *bytes.Buffer, attrs map[string]string) {
	keys := make([]string, 0, len(attrs))
	for k, v := range attrs {
		if k == "" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteString(`="`)
		b.WriteString(xmlEscape(attrs[k]))
		b.WriteByte('"')
	}
}

func xmlEscape(v string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(v))
	return b.String()
}

func boolAttr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func intAttr(v int) string {
	return fmt.Sprint(v)
}
