// Command serve is a minimal dev server for testing Lux WASM builds.
//
// Usage:
//
//	GOOS=js GOARCH=wasm go build -o web/wasm/app.wasm ./examples/counter
//	cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" web/wasm/
//	go run ./web/wasm/serve.go
//
// Then open http://localhost:8080 in a WebGPU-capable browser.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dir := flag.String("dir", "", "directory to serve (default: directory containing this file)")
	flag.Parse()

	serveDir := *dir
	if serveDir == "" {
		exe, err := os.Executable()
		if err == nil {
			serveDir = filepath.Dir(exe)
		} else {
			serveDir = "."
		}
	}

	if _, err := os.Stat(filepath.Join(serveDir, "index.html")); os.IsNotExist(err) {
		serveDir = "web/wasm"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".wasm" {
			w.Header().Set("Content-Type", "application/wasm")
		}
		http.FileServer(http.Dir(serveDir)).ServeHTTP(w, r)
	})

	fmt.Printf("Serving %s on http://localhost%s\n", serveDir, *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
