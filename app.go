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
	Writable bool
}

type app struct {
	ctx      context.Context
	root     string
	cancel   func()
	server   *tsnet.Server
	handler  *FileServer
	funnel   bool
	writable bool
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

	handler, err := NewFileServer(args.Root, args.Writable)
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

func (a *app) Close() {
	defer a.cancel()
}

func (a *app) Run() error {
	log.Printf("Starting file server on %s", a.handler.Root())
	defer a.cancel()
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
	if err := httpServer.Serve(ln); err != nil {
		return err
	}
	return nil
}

type FileServer struct {
	root     string
	writable bool
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

<script>
async function upload() {
	const input = document.getElementById("file")
	for (const file of input.files) {
	    const {name} = file
	    const url = window.location.toString() + "/" + name
	    const xhr = new XMLHttpRequest()
	    xhr.open('POST', url, true)
	    xhr.upload.onprogress = function(event) {
	      if (event.lengthComputable) {
	          const percentComplete = (event.loaded / event.total) * 100;
	          document.getElementById("status").innerText = (name + ": " + percentComplete.toFixed(2) + "%");
	      }
		};
	    console.log(file)
	    xhr.send(file)
	}
	document.getElementById("status").innerText = "Finished"
}
</script>

<input type="file" id="file" multiple /><button onclick="upload()">Upload</button>
<p id="status"></p>
<ul>
`

func (f *FileServer) WriteHTMLPrelude(w io.Writer) {
	fmt.Fprintf(w, "%s", HTML_PRELUDE)
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
			for _, entry := range entries {
				fmt.Fprintf(w, "<li><a href=\"%s\">%s</a></li>", r.URL.JoinPath(entry.Name()), entry.Name())
			}
		} else {
			f, err := os.Open(item)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "can't open file to be read: %s", err.Error())
				return
			}
			w.Header().Add("Content-Length", fmt.Sprintf("%d", info.Size()))
			w.Header().Add("Content-Type", "application/octet-stream")
			defer f.Close()
			buf := make([]byte, 1024*1024)
			io.CopyBuffer(w, f, buf)
		}
	}
	if r.Method == http.MethodPost && !f.writable {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "i'm afraid i can't do that")
		return
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
		buf := make([]byte, 1024*1024)
		io.CopyBuffer(f, r.Body, buf)

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
