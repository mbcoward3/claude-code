package main

import (
	"embed"
	"io/fs"
)

// Browser-side assets are compiled into the binary so the tool ships as a single
// artifact with no CDN or network dependency at runtime.
//
//go:embed assets/sdk.js assets/chrome.css assets/shell.html assets/tailwind.js assets/daisyui.css assets/playbook.md
var assetFS embed.FS

// asset reads an embedded asset by name (relative to the assets/ directory).
func asset(name string) ([]byte, error) {
	return assetFS.ReadFile("assets/" + name)
}

// assetString is asset but returns a string, panicking on missing assets since
// they are compiled in and their absence is a build error, not a runtime one.
func mustAsset(name string) string {
	b, err := asset(name)
	if err != nil {
		panic("missing embedded asset: " + name)
	}
	return string(b)
}

// contentType maps an asset extension to a MIME type for static serving.
func contentType(name string) string {
	switch {
	case hasSuffix(name, ".js"):
		return "application/javascript; charset=utf-8"
	case hasSuffix(name, ".css"):
		return "text/css; charset=utf-8"
	case hasSuffix(name, ".html"):
		return "text/html; charset=utf-8"
	case hasSuffix(name, ".md"):
		return "text/markdown; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// staticAssets lists the asset names served under /assets/.
func staticAssets() []string {
	entries, _ := fs.ReadDir(assetFS, "assets")
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}
