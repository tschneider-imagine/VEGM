package runtime

import (
	"fmt"
	"sort"
	"strings"
)

func RenderTemplate(tmpl string, ns map[string]string, request map[string]string, state map[string]any) string {
	out := tmpl
	keys := make([]string, 0, len(ns))
	for k := range ns {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{ns.%s}}", k), ns[k])
	}
	for k, v := range request {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{request.%s}}", k), v)
	}
	for k, v := range state {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{state.%s}}", k), fmt.Sprint(v))
	}
	return out
}
