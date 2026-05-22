package runtime

import (
	"net/http"
	"testing"

	packpkg "github.com/tschneider-imagine/VEGM/pack"
)

func TestRegisteredHosts_StrictModeAllowsEnabledHost(t *testing.T) {
	srv := newTestServer(t)
	srv.state.RegisteredHosts = []packpkg.RegisteredHost{
		{HostID: "HOST-001", Role: "owner", Enabled: true, AllowedClasses: []string{"comm", "audio", "cabinet"}},
	}

	rr := postG2S(t, srv, soapBody("audioMuteOn", "<hostId>HOST-001</hostId><sessionId>S-REG-OK</sessionId>"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected enabled registered host to pass, got %d body=%s", rr.Code, rr.Body.String())
	}
	if srv.state.AudioState != "muted" {
		t.Fatalf("expected audio muted, got %q", srv.state.AudioState)
	}
}

func TestRegisteredHosts_StrictModeRejectsDisabledHost(t *testing.T) {
	srv := newTestServer(t)
	srv.state.RegisteredHosts = []packpkg.RegisteredHost{
		{HostID: "HOST-001", Role: "owner", Enabled: false, AllowedClasses: []string{"comm", "audio", "cabinet"}},
	}

	rr := postG2S(t, srv, soapBody("audioMuteOn", "<hostId>HOST-001</hostId><sessionId>S-REG-DISABLED</sessionId>"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected disabled registered host to be rejected, got %d body=%s", rr.Code, rr.Body.String())
	}
	if srv.state.AudioState != "normal" {
		t.Fatalf("expected audio unchanged, got %q", srv.state.AudioState)
	}
}

func TestRegisteredHosts_StrictModeRejectsUnknownWhenRegisteredHostsPresent(t *testing.T) {
	srv := newTestServer(t)
	srv.state.RegisteredHosts = []packpkg.RegisteredHost{
		{HostID: "HOST-001", Role: "owner", Enabled: true, AllowedClasses: []string{"comm", "audio", "cabinet"}},
	}

	rr := postG2S(t, srv, soapBody("audioMuteOn", "<hostId>HOST-404</hostId><sessionId>S-REG-UNKNOWN</sessionId>"))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected unknown host to be rejected, got %d body=%s", rr.Code, rr.Body.String())
	}
}
