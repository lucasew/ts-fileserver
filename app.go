package fileserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"tailscale.com/tsnet"
)

type AppParams struct {
	Root     string
	Ctx      context.Context
	StateDir string
	Name     string
	Funnel   bool
}

type app struct {
	ctx     context.Context
	root    string
	cancel  func()
	server  *tsnet.Server
	handler *FileServer
	funnel  bool
}

var (
	ErrNotADir = errors.New("not a directory")
)

func NewApp(args AppParams) (*app, error) {
	if args.Ctx == nil {
		args.Ctx = context.Background()
	}
	server := new(tsnet.Server)
	if args.Name == "" {
		args.Name = "ts-fileserver"
	}
	server.Hostname = args.Name
	if args.StateDir != "" {
		if err := os.MkdirAll(args.StateDir, 0700); err != nil {
			return nil, err
		}
		server.Dir = args.StateDir
	}

	ctx, cancel := context.WithCancel(args.Ctx)

	handler, err := NewFileServer(args.Root)
	if err != nil {
		return nil, err
	}

	return &app{
		ctx:     ctx,
		cancel:  cancel,
		handler: handler,
		root:    args.Root,
		server:  server,
		funnel:  args.Funnel,
	}, nil
}

func (a *app) Run() error {
	log.Printf("Starting file server on %s", a.handler.Root())
	var ln net.Listener
	var err error
	if a.funnel {
		if ln, err = a.server.ListenFunnel("tcp", ":443"); err != nil {
			return err
		}
	} else {
		if ln, err = a.server.ListenTLS("tcp", ":443"); err != nil {
			return err
		}
	}
	for _, domain := range a.server.CertDomains() {
		log.Printf("To use it please access: https://%s", domain)
	}
	httpServer := http.Server{Handler: a.handler}
	return httpServer.Serve(ln)
}

type FileServer struct {
	root string
}

const HTML_PRELUDE = `
<!DOCTYPE html>
<html lang="en">
    <head>
        <meta charset="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    	<title>ts-fileserver</title>
		<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/sakura.css/css/sakura.css" type="text/css">
    </head>
<body>
`

func (f *FileServer) WriteHTMLPrelude(w io.Writer) {
	fmt.Fprintf(w, HTML_PRELUDE)
}

// ServeHTTP implements http.Handler.
func (f *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s %s", r.Method, r.RemoteAddr, r.URL.Path)
	w.WriteHeader(200)

	item := path.Join(f.root, r.URL.Path)
	if !strings.HasPrefix(item, f.root) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "nice try!")
		return
	}
	if r.Method == http.MethodGet {
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
			fmt.Fprintf(w, "<ul>")
			for _, entry := range entries {
				fmt.Fprintf(w, "<li><a href=\"%s\">%s</a></li>", r.URL.JoinPath(entry.Name()), entry.Name())
			}
			fmt.Fprintf(w, "</ul>")
		} else {
			f, err := os.Open(item)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "can't open file to be read: %s", err.Error())
				return
			}
			w.Header().Add("Content-Length", fmt.Sprintf("%d", info.Size()))
			defer f.Close()
			io.Copy(w, f)
		}
	}
	if r.Method == http.MethodPost {
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
		f, err := os.Create(item)
		defer f.Close()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "can't create file: %s", err.Error())
		}
		io.Copy(f, r.Body)

	}

}

func (f *FileServer) Root() string {
	return f.root
}

func NewFileServer(root string) (*FileServer, error) {
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
	return &FileServer{root: root}, nil
}
