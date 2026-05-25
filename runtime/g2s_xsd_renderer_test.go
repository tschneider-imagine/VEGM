package runtime

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
)

func TestG2SXMLDefaultsToLegacyMode(t *testing.T) {
	cfg := &Config{
		InstanceID: "vegm-default-mode",
		HostID:     "HOST-001",
		EGMID:      "EGM-001",
		EGMEndpoint: EGMEndpointConfig{
			Scheme: "http",
			BindIP: "127.0.0.1",
			Host:   "127.0.0.1",
			Port:   18443,
			Path:   "/g2s",
		},
		Listen:   ListenConfig{Host: "127.0.0.1", Port: 18443},
		Security: SecurityConfig{TrustMode: "plaintext_lab"},
		Logging:  LoggingConfig{Dir: t.TempDir()},
		Control:  ControlConfig{Bind: "127.0.0.1:0"},
		PackFile: "../example.pack.json",
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("validate config: %v", err)
	}
	if cfg.G2SXML.Mode != G2SXMLModeLegacy {
		t.Fatalf("mode = %q, want %q", cfg.G2SXML.Mode, G2SXMLModeLegacy)
	}
	if cfg.G2SXML.Namespace != G2SDefaultNamespace {
		t.Fatalf("namespace = %q, want %q", cfg.G2SXML.Namespace, G2SDefaultNamespace)
	}
	if cfg.G2SXML.EGMLocation != "127.0.0.1:18443" {
		t.Fatalf("egm location = %q", cfg.G2SXML.EGMLocation)
	}
}

func TestRenderXSDCommsOnLine(t *testing.T) {
	s := newXSDRendererTestServer()
	got := s.renderCommsOnline("session-ignored-in-xsd-pass1")
	assertWellFormedXML(t, got)
	assertContainsAll(t, got,
		`<g2s:g2sMessage xmlns:g2s="`+G2SDefaultNamespace+`">`,
		`<g2s:g2sBody`,
		`hostId="HOST-001"`,
		`egmId="EGM-001"`,
		`dateTimeSent="`,
		`<g2s:communications><g2s:commsOnLine`,
		`egmLocation="192.168.10.161:18443"`,
		`equipmentType="G2S_egm"`,
		`deviceReset="false"`,
		`deviceChanged="false"`,
		`subscriptionLost="false"`,
		`metersReset="false"`,
	)
}

func TestRenderXSDGetDescriptor(t *testing.T) {
	s := newXSDRendererTestServer()
	got := s.renderGetDescriptor("session-ignored-in-xsd-pass1")
	assertWellFormedXML(t, got)
	assertContainsAll(t, got,
		`<g2s:communications><g2s:getDescriptor`,
		`includeOwners="true"`,
		`includeConfigs="true"`,
		`includeGuests="true"`,
		`includeOthers="true"`,
	)
}

func TestRenderXSDSetKeepAliveUsesIntervalAttribute(t *testing.T) {
	s := newXSDRendererTestServer()
	got := s.renderSetKeepAlive("session-ignored-in-xsd-pass1")
	assertWellFormedXML(t, got)
	assertContainsAll(t, got,
		`<g2s:communications><g2s:setKeepAlive`,
		`interval="5000"`,
	)
	if strings.Contains(got, "keepAliveIntervalMS") {
		t.Fatalf("xsd setKeepAlive should use interval attribute, got %s", got)
	}
}

func TestRenderXSDKeepAlive(t *testing.T) {
	s := newXSDRendererTestServer()
	got := s.renderKeepAlive("session-ignored-in-xsd-pass1")
	assertWellFormedXML(t, got)
	assertContainsAll(t, got,
		`<g2s:g2sMessage`,
		`<g2s:g2sBody`,
		`<g2s:communications><g2s:keepAlive`,
		`hostId="HOST-001"`,
		`egmId="EGM-001"`,
	)
}

func newXSDRendererTestServer() *Server {
	return &Server{cfg: &Config{
		InstanceID: "vegm-xsd-renderer-test",
		HostID:     "HOST-001",
		EGMID:      "EGM-001",
		EGMEndpoint: EGMEndpointConfig{
			Scheme: "http",
			BindIP: "192.168.10.161",
			Host:   "192.168.10.161",
			Port:   18443,
			Path:   "/g2s",
		},
		G2SXML: G2SXMLConfig{
			Mode:        G2SXMLModeXSDMessage,
			Namespace:   G2SDefaultNamespace,
			EGMLocation: "192.168.10.161:18443",
		},
		SessionEngine: SessionEngineConfig{KeepAliveIntervalMS: 5000},
	}}
}

func assertContainsAll(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered xml missing %q\nxml: %s", want, got)
		}
	}
}

func assertWellFormedXML(t *testing.T, got string) {
	t.Helper()
	dec := xml.NewDecoder(bytes.NewReader([]byte(got)))
	for {
		if _, err := dec.Token(); err != nil {
			if err.Error() == "EOF" {
				return
			}
			t.Fatalf("rendered XML is not well-formed: %v\nxml: %s", err, got)
		}
	}
}
