package runtime

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLatestEvidencePrefersResponseMatchingLatestRequestType(t *testing.T) {
	dir := t.TempDir()

	writePayloadForTest(t, dir, "20260101T000000.000000000Z_outbound_request_keepAlive.xml", `<Envelope><Body><keepAlive/></Body></Envelope>`, -4*time.Minute)
	writePayloadForTest(t, dir, "20260101T000001.000000000Z_outbound_response_keepAlive.xml", `<Envelope><Body><g2sResponse><keepAliveAck/></g2sResponse></Body></Envelope>`, -3*time.Minute)
	writePayloadForTest(t, dir, "20260101T000002.000000000Z_outbound_request_commsOnLine.xml", `<Envelope><Body><commsOnLine/></Body></Envelope>`, -2*time.Minute)
	writePayloadForTest(t, dir, "20260101T000003.000000000Z_outbound_response_commsOnLine.xml", `<Envelope><Body><g2sResponse><commsOnLineAck/></g2sResponse></Body></Envelope>`, -1*time.Minute)

	request := latestPayload(dir, "outbound_request")
	if request.MessageType != "commsOnLine" {
		t.Fatalf("latest request type = %q, want commsOnLine", request.MessageType)
	}

	response := latestPayloadForMessageType(dir, "outbound_response", request.MessageType)
	if response.MessageType != "commsOnLine" {
		t.Fatalf("matched response type = %q, want commsOnLine", response.MessageType)
	}
	if response.Name != "20260101T000003.000000000Z_outbound_response_commsOnLine.xml" {
		t.Fatalf("matched response name = %q", response.Name)
	}
}

func TestLatestEvidenceFallsBackWhenMatchingResponseMissing(t *testing.T) {
	dir := t.TempDir()

	writePayloadForTest(t, dir, "20260101T000000.000000000Z_outbound_response_keepAlive.xml", `<Envelope><Body><g2sResponse><keepAliveAck/></g2sResponse></Body></Envelope>`, -2*time.Minute)
	writePayloadForTest(t, dir, "20260101T000001.000000000Z_outbound_request_commsOnLine.xml", `<Envelope><Body><commsOnLine/></Body></Envelope>`, -1*time.Minute)

	request := latestPayload(dir, "outbound_request")
	if request.MessageType != "commsOnLine" {
		t.Fatalf("latest request type = %q, want commsOnLine", request.MessageType)
	}

	matched := latestPayloadForMessageType(dir, "outbound_response", request.MessageType)
	if matched.Path != "" {
		t.Fatalf("expected no matching commsOnLine response, got %q", matched.Name)
	}

	fallback := latestPayload(dir, "outbound_response")
	if fallback.MessageType != "keepAlive" {
		t.Fatalf("fallback response type = %q, want keepAlive", fallback.MessageType)
	}
}

func writePayloadForTest(t *testing.T, dir, name, content string, age time.Duration) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write payload %s: %v", name, err)
	}
	when := time.Now().Add(age)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatalf("chtimes payload %s: %v", name, err)
	}
}
