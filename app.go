package fileserver

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"

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

	handler, err := NewFileServer(args.Root, args.Writable)
	if err != nil {
		cancel()
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
