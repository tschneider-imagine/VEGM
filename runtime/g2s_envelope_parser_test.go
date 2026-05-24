package runtime

import "testing"

func TestParseLegacySOAPKeepAliveAck(t *testing.T) {
	raw := []byte(`<soapenv:Envelope xmlns:soapenv="` + SOAP11Namespace + `" xmlns:g2s="urn:g2s:lab"><soapenv:Body><g2s:keepAliveAck><g2s:hostId>HOST-001</g2s:hostId><g2s:egmId>EGM-001</g2s:egmId><g2s:sessionId>S-1</g2s:sessionId></g2s:keepAliveAck></soapenv:Body></soapenv:Envelope>`)
	parsed, err := ParseG2SEnvelope(raw)
	if err != nil {
		t.Fatalf("parse legacy soap: %v", err)
	}
	if parsed.RootKind != "legacy" {
		t.Fatalf("RootKind = %q, want legacy", parsed.RootKind)
	}
	if parsed.OperationName != "keepAliveAck" {
		t.Fatalf("OperationName = %q", parsed.OperationName)
	}
	if parsed.HostID != "HOST-001" || parsed.EGMID != "EGM-001" || parsed.SessionID != "S-1" {
		t.Fatalf("identity parse failed: %+v", parsed)
	}
}

func TestParseXSDKeepAliveAck(t *testing.T) {
	raw := []byte(`<g2s:g2sMessage xmlns:g2s="` + G2SDefaultNamespace + `"><g2s:g2sBody hostId="HOST-001" egmId="EGM-001" dateTimeSent="2026-05-24T00:00:00Z"><g2s:communications><g2s:keepAliveAck sessionId="S-2" /></g2s:communications></g2s:g2sBody></g2s:g2sMessage>`)
	parsed, err := ParseG2SEnvelope(raw)
	if err != nil {
		t.Fatalf("parse xsd keepAliveAck: %v", err)
	}
	if parsed.RootKind != "g2sMessage" {
		t.Fatalf("RootKind = %q, want g2sMessage", parsed.RootKind)
	}
	if parsed.ClassName != "communications" {
		t.Fatalf("ClassName = %q", parsed.ClassName)
	}
	if parsed.OperationName != "keepAliveAck" {
		t.Fatalf("OperationName = %q", parsed.OperationName)
	}
	if parsed.HostID != "HOST-001" || parsed.EGMID != "EGM-001" || parsed.SessionID != "S-2" {
		t.Fatalf("identity parse failed: %+v", parsed)
	}
}

func TestParseXSDDescriptorList(t *testing.T) {
	raw := []byte(`<g2s:g2sMessage xmlns:g2s="` + G2SDefaultNamespace + `"><g2s:g2sBody hostId="HOST-001" egmId="EGM-001" dateTimeSent="2026-05-24T00:00:00Z"><g2s:communications><g2s:descriptorList /></g2s:communications></g2s:g2sBody></g2s:g2sMessage>`)
	parsed, err := ParseG2SEnvelope(raw)
	if err != nil {
		t.Fatalf("parse xsd descriptorList: %v", err)
	}
	if parsed.OperationName != "descriptorList" {
		t.Fatalf("OperationName = %q", parsed.OperationName)
	}
}

func TestParseG2SAck(t *testing.T) {
	raw := []byte(`<g2s:g2sMessage xmlns:g2s="` + G2SDefaultNamespace + `"><g2s:g2sAck hostId="HOST-001" egmId="EGM-001" dateTimeSent="2026-05-24T00:00:00Z" /></g2s:g2sMessage>`)
	parsed, err := ParseG2SEnvelope(raw)
	if err != nil {
		t.Fatalf("parse g2sAck: %v", err)
	}
	if parsed.RootKind != "g2sAck" {
		t.Fatalf("RootKind = %q", parsed.RootKind)
	}
	if parsed.OperationName != "g2sAck" {
		t.Fatalf("OperationName = %q", parsed.OperationName)
	}
}

func TestParseG2SMessageReturnsOperationName(t *testing.T) {
	raw := []byte(`<g2s:g2sMessage xmlns:g2s="` + G2SDefaultNamespace + `"><g2s:g2sBody hostId="HOST-001" egmId="EGM-001" dateTimeSent="2026-05-24T00:00:00Z"><g2s:communications><g2s:setKeepAliveAck /></g2s:communications></g2s:g2sBody></g2s:g2sMessage>`)
	parsed, err := ParseG2SMessage(raw)
	if err != nil {
		t.Fatalf("parse g2s message: %v", err)
	}
	if parsed.RootLocalName != "setKeepAliveAck" {
		t.Fatalf("RootLocalName = %q", parsed.RootLocalName)
	}
}

func TestExpectedAckRootRemainsStrict(t *testing.T) {
	if got := expectedAckRoot("keepAlive"); got != "keepAliveAck" {
		t.Fatalf("keepAlive expected ack = %q", got)
	}
	if got := expectedAckRoot("commsOnLine"); got != "commsOnLineAck" {
		t.Fatalf("commsOnLine expected ack = %q", got)
	}
}
