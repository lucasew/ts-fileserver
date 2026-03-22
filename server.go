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
)

type FileServer struct {
	root     string
	writable bool
}

// ServeHTTP implements http.Handler.
func (f *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s %s", r.Method, r.RemoteAddr, r.URL.Path)
	item := path.Join(f.root, r.URL.Path)
	if !strings.HasPrefix(item, f.root) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "nice try!")
		return
	}
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	if r.URL.Path == "/installHook.js.map" || r.URL.Path == "/favicon.ico" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "these common routes are ignored")
		return
	}
	switch r.Method {
	case http.MethodGet:
		f.handleGet(w, r, item)
	case http.MethodPost:
		f.handlePost(w, r, item)
	}
}

func (f *FileServer) handleGet(w http.ResponseWriter, r *http.Request, item string) {
	info, err := os.Stat(item)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "can't stat item: %s", err.Error())
		return
	}
	if info.IsDir() {
		entries, err := os.ReadDir(item)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "can't list folder entries: %s", err.Error())
			return
		}
		f.WriteHTMLPrelude(w)
		fmt.Fprintf(w, "<h1>Files in %s</h1>", item)
		for _, entry := range entries {
			fmt.Fprintf(w, "<li><a href=\"%s\">%s</a></li>", r.URL.JoinPath(entry.Name()), entry.Name())
		}
	} else {
		file, err := os.Open(item)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "can't open file to be read: %s", err.Error())
			return
		}
		w.Header().Add("Content-Length", fmt.Sprintf("%d", info.Size()))
		defer file.Close()
		buf := make([]byte, 1024*1024)
		io.CopyBuffer(w, file, buf)
	}
}

func (f *FileServer) handlePost(w http.ResponseWriter, r *http.Request, item string) {
	if !f.writable {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "i'm afraid i can't do that")
		return
	}
	info, err := os.Stat(item)
	if info != nil && info.IsDir() {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "path should not be a existing folder")
		return
	}
	if err := os.MkdirAll(path.Dir(item), os.ModePerm); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "can't create parent directory: %s", err.Error())
		return
	}
	file, err := os.Create(item)
	defer file.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "can't create file: %s", err.Error())
	}
	buf := make([]byte, 1024*1024)
	io.CopyBuffer(file, r.Body, buf)
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
