package httpapi

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// notBuiltPage is served when the binary carries no dashboard build. The API
// works regardless, so this says what is missing rather than failing.
const notBuiltPage = `<!doctype html>
<meta charset="utf-8">
<title>Croffle Server</title>
<style>
  body { font: 15px/1.6 system-ui, sans-serif; margin: 10vh auto; max-width: 34rem; padding: 0 1rem; }
  code { background: #eee9e3; padding: 0.1rem 0.3rem; border-radius: 3px; }
</style>
<h1>Croffle Server</h1>
<p>This build carries no dashboard. The sync API is running.</p>
<p>To include it, build the web project before the binary:</p>
<pre><code>cd apps/server/web &amp;&amp; pnpm install &amp;&amp; pnpm build
cd .. &amp;&amp; make build</code></pre>
<p><a href="health">health</a></p>
`

// spaHandler serves the embedded dashboard, falling back to index.html so the
// single-page app can own its own paths.
//
// It is mounted last and only under the base path, so it can never shadow
// /health, /auth, /sync or /dash: those patterns are more specific and
// net/http prefers them.
func (s *server) spaHandler(assets fs.FS) http.HandlerFunc {
	index, indexErr := fs.ReadFile(assets, "index.html")
	fileServer := http.FileServer(http.FS(assets))

	return func(w http.ResponseWriter, r *http.Request) {
		if indexErr != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(notBuiltPage))
			return
		}

		relative := s.stripBase(r.URL.Path)

		// A request that looks like a file either exists or is a 404; only
		// routes fall back to index.html. Serving the app for a missing asset
		// would turn a broken script tag into a confusing blank page.
		if ext := path.Ext(relative); ext != "" {
			if _, err := fs.Stat(assets, strings.TrimPrefix(relative, "/")); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					writeError(w, http.StatusNotFound, "not_found", "no such file")
					return
				}
			}
			stripped := http.StripPrefix(s.assetPrefix(), fileServer)
			stripped.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// The dashboard is behind a login and shows account data; a cache in
		// between must not keep it.
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(index)
	}
}

// stripBase turns a request path into one relative to the mount point.
func (s *server) stripBase(urlPath string) string {
	if s.basePath == "/" {
		return urlPath
	}
	return "/" + strings.TrimPrefix(strings.TrimPrefix(urlPath, s.basePath), "/")
}

func (s *server) assetPrefix() string {
	if s.basePath == "/" {
		return ""
	}
	return s.basePath
}
