package main

import (
	"fmt"
	"os"
	"strings"
)

// injectedHead is the markup inserted into every served artifact. It wires up the
// local (embedded) Tailwind + DaisyUI, the chrome styles, and the annotation SDK,
// and exposes the session key to the browser. All URLs are same-origin and local.
const injectedHeadTmpl = `
<!-- plan-ui: injected runtime (local, no CDN) -->
<script>window.__PLAN_UI__ = { key: %q };</script>
<script src="/assets/tailwind.js"></script>
<link rel="stylesheet" href="/assets/daisyui.css">
<link rel="stylesheet" href="/assets/chrome.css">
<script defer src="/assets/sdk.js"></script>
`

// transformArtifact reads the agent-authored HTML file and injects the plan-ui
// runtime so the saved file still renders standalone but gains annotation,
// live-reload, and the layout gate when served through plan-ui.
func transformArtifact(canonical, key string) ([]byte, error) {
	raw, err := os.ReadFile(canonical)
	if err != nil {
		return nil, err
	}
	html := string(raw)
	inject := fmt.Sprintf(injectedHeadTmpl, key)

	// Prefer injecting just after <head>; fall back to before </head>, then
	// wrap bare fragments in a minimal document.
	if idx := headOpenIndex(html); idx >= 0 {
		return []byte(html[:idx] + inject + html[idx:]), nil
	}
	if idx := strings.Index(strings.ToLower(html), "</head>"); idx >= 0 {
		return []byte(html[:idx] + inject + html[idx:]), nil
	}
	// No <head>: build a document around the content.
	wrapped := "<!doctype html><html><head><meta charset=\"utf-8\">" + inject +
		"</head><body>" + html + "</body></html>"
	return []byte(wrapped), nil
}

// headOpenIndex returns the byte offset just after the opening <head ...> tag,
// or -1 if there is none.
func headOpenIndex(html string) int {
	lower := strings.ToLower(html)
	i := strings.Index(lower, "<head")
	if i < 0 {
		return -1
	}
	// Advance to the end of the opening tag.
	end := strings.IndexByte(lower[i:], '>')
	if end < 0 {
		return -1
	}
	return i + end + 1
}
