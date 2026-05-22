package runtime

import (
	"fmt"
	"net/http"
	"strings"
)

type bindingCheckResult struct {
	OK         bool
	StatusCode int
	Message    string
}

func validateG2SBindingRequest(r *http.Request) bindingCheckResult {
	if r.Method != http.MethodPost {
		return bindingCheckResult{OK: false, StatusCode: http.StatusMethodNotAllowed, Message: "method not allowed"}
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if contentType == "" {
		return bindingCheckResult{OK: false, StatusCode: http.StatusUnsupportedMediaType, Message: "content-type is required"}
	}
	mediaType := strings.TrimSpace(strings.Split(contentType, ";")[0])
	switch mediaType {
	case "text/xml", "application/xml", "application/soap+xml":
		return bindingCheckResult{OK: true}
	default:
		return bindingCheckResult{OK: false, StatusCode: http.StatusUnsupportedMediaType, Message: fmt.Sprintf("unsupported content-type %q", contentType)}
	}
}
