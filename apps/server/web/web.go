// Package web embeds the built dashboard into the binary.
//
// The Go file lives next to the Vue project because //go:embed cannot reach
// outside its own directory, and the build output has to sit below the package
// that embeds it.
package web

import (
	"embed"
	"io/fs"
)

// The "all:" prefix includes the tracked .gitkeep placeholder, which is what
// makes this compile when the dashboard has not been built. Without a tracked
// file here, every `go build` would require a Node toolchain first.
//
//go:embed all:embedded
var embedded embed.FS

// FS returns the dashboard files, rooted so that index.html is at the top.
func FS() (fs.FS, error) {
	return fs.Sub(embedded, "embedded")
}

// Built reports whether a dashboard build is present. A binary without one
// still serves the API; only the HTML is missing.
func Built() bool {
	entries, err := embedded.ReadDir("embedded")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.Name() == "index.html" {
			return true
		}
	}
	return false
}
