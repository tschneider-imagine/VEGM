package runtime

import "testing"

func TestParseG2SMessage_ValidSOAP11(t *testing.T) {
	body := []byte(`<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:g2s="urn:test:g2s"><soapenv:Body><g2s:commsOnLine><g2s:hostId>HOST-001</g2s:hostId><g2s:sessionId>S-1</g2s:sessionId></g2s:commsOnLine></soapenv:Body></soapenv:Envelope>`)
	msg, err := ParseG2SMessage(body)
	if err != nil {
		t.Fatalf("ParseG2SMessage failed: %v", err)
	}
	if msg.RootLocalName != "commsOnLine" {
		t.Fatalf("expected commsOnLine, got %q", msg.RootLocalName)
	}
	if msg.SOAPNamespace != SOAP11Namespace {
		t.Fatalf("expected SOAP 1.1 namespace, got %q", msg.SOAPNamespace)
	}
	if msg.Fields["hostId"] != "HOST-001" || msg.Fields["sessionId"] != "S-1" {
		t.Fatalf("unexpected parsed fields: %#v", msg.Fields)
	}
}

func TestParseG2SMessage_RejectsMissingEnvelope(t *testing.T) {
	_, err := ParseG2SMessage([]byte(`<Body><commsOnLine/></Body>`))
	if err == nil {
		t.Fatalf("expected missing envelope error")
	}
}

func TestParseG2SMessage_RejectsUnsupportedSOAPNamespace(t *testing.T) {
	body := []byte(`<x:Envelope xmlns:x="urn:not-soap"><x:Body><commsOnLine/></x:Body></x:Envelope>`)
	_, err := ParseG2SMessage(body)
	if err == nil {
		t.Fatalf("expected unsupported SOAP namespace error")
	}
}

func TestParseG2SMessage_RejectsMissingBodyMessage(t *testing.T) {
	body := []byte(`<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"><soapenv:Body></soapenv:Body></soapenv:Envelope>`)
	_, err := ParseG2SMessage(body)
	if err == nil {
		t.Fatalf("expected missing body message error")
	}
}
