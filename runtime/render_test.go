package runtime

import "testing"

func TestRenderTemplate_ReplacesNamespacesRequestAndState(t *testing.T) {
	tmpl := "<x xmlns:g2s='{{ns.g2s}}'><host>{{request.hostId}}</host><state>{{state.session_state}}</state></x>"
	ns := map[string]string{"g2s": "urn:test:g2s"}
	req := map[string]string{"hostId": "HOST-001"}
	state := map[string]any{"session_state": "online"}
	got := RenderTemplate(tmpl, ns, req, state)
	want := "<x xmlns:g2s='urn:test:g2s'><host>HOST-001</host><state>online</state></x>"
	if got != want {
		t.Fatalf("unexpected rendered template\nwant: %s\ngot:  %s", want, got)
	}
}
