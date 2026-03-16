package fileserver

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/lucasew/ts-fileserver/internal/reporter"
)

type FileServer struct {
	root     string
	writable bool
}

func (f *FileServer) WriteHTMLPrelude(w io.Writer) {
	_, _ = fmt.Fprintf(w, "%s", HTML_PRELUDE)
}

// ServeHTTP implements http.Handler.
func (f *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s %s", r.Method, r.RemoteAddr, r.URL.Path)
	item := path.Join(f.root, r.URL.Path)
	if !strings.HasPrefix(item, f.root) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, "nice try!")
		return
	}
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	if r.URL.Path == "/installHook.js.map" || r.URL.Path == "/favicon.ico" {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprintf(w, "these common routes are ignored")
		return
	}
	if r.Method == http.MethodGet {
		info, err := os.Stat(item)
		if err != nil {
			reporter.ReportError(fmt.Errorf("can't stat item: %w", err), map[string]any{"path": item})
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, "can't stat item: %s", err.Error())
			return
		}
		if info.IsDir() {
			entries, err := os.ReadDir(item)
			if err != nil {
				reporter.ReportError(fmt.Errorf("can't list folder entries: %w", err), map[string]any{"path": item})
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprintf(w, "can't list folder entries: %s", err.Error())
				return
			}
			f.WriteHTMLPrelude(w)
			_, _ = fmt.Fprintf(w, "<h1>Files in %s</h1>", item)
			for _, entry := range entries {
				_, _ = fmt.Fprintf(w, "<li><a href=\"%s\">%s</a></li>", r.URL.JoinPath(entry.Name()), entry.Name())
			}
		} else {
			file, err := os.Open(item)
			if err != nil {
				reporter.ReportError(fmt.Errorf("can't open file to be read: %w", err), map[string]any{"path": item})
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprintf(w, "can't open file to be read: %s", err.Error())
				return
			}
			w.Header().Add("Content-Length", fmt.Sprintf("%d", info.Size()))
			// w.Header().Add("Content-Type", "application/octet-stream")
			defer func() { _ = file.Close() }()
			buf := make([]byte, 1024*1024)
			_, _ = io.CopyBuffer(w, file, buf)
		}
	}
	if r.Method == http.MethodPost && !f.writable {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprintf(w, "i'm afraid i can't do that")
		return
	}
	if r.Method == http.MethodPost {
		info, _ := os.Stat(item)
		if info != nil && info.IsDir() {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, "path should not be a existing folder")
			return
		}
		if err := os.MkdirAll(path.Dir(item), os.ModePerm); err != nil {
			reporter.ReportError(fmt.Errorf("can't create parent directory: %w", err), map[string]any{"path": item})
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, "can't create parent directory: %s", err.Error())
			return
		}
		file, err := os.Create(item)
		if err != nil {
			reporter.ReportError(fmt.Errorf("can't create file: %w", err), map[string]any{"path": item})
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, "can't create file: %s", err.Error())
			return
		}
		defer func() { _ = file.Close() }()
		buf := make([]byte, 1024*1024)
		_, _ = io.CopyBuffer(file, r.Body, buf)
	}
}

func (f *FileServer) Root() string {
	return f.root
}

func NewFileServer(root string, writable bool) (*FileServer, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !rootInfo.IsDir() {
		return nil, ErrNotADir
	}
	return &FileServer{root: root, writable: writable}, nil
}
