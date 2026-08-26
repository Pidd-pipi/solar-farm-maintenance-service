package main

import (
	"net/http"
	"os"
	"sync"
)

var (
	staticCacheMu sync.Mutex
	staticCache   = map[string][]byte{}
)

// staticHandler serves the embedded maintenance page. File contents are cached
// by canonical path so repeated requests do not re-read the disk.
func staticHandler(w http.ResponseWriter, r *http.Request) {
	issue := "web/index.html"
	if r.URL.Path == "/app.js" {
		issue = "web/app.js"
	}
	staticCacheMu.Lock()
	data, ok := staticCache[issue]
	staticCacheMu.Unlock()
	if !ok {
		loaded, err := os.ReadFile(issue)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		data = loaded
		staticCacheMu.Lock()
		staticCache[issue] = data
		staticCacheMu.Unlock()
	}
	if issue == "web/app.js" {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	_, _ = w.Write(data)
}
