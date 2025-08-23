package main

import (
	"flag"
	"log"

	"github.com/davecgh/go-spew/spew"
	fileserver "github.com/lucasew/ts-fileserver"
)

func main() {
	var params fileserver.AppParams

	flag.StringVar(&params.Root, "r", ".", "Which folder to expose")
	flag.StringVar(&params.StateDir, "s", "", "Where to store Tailscale state")
	flag.StringVar(&params.Name, "n", "ts-fileserver", "Hostname of this Tailscale node")
	flag.BoolVar(&params.Funnel, "f", false, "Expose it to the Internet?")
	flag.BoolVar(&params.Writable, "w", false, "Are users able to write files?")
	flag.Parse()

	spew.Dump("args: ", params)

	app, err := fileserver.NewApp(params)
	if err != nil {
		log.Fatalf("failed to initialize application: %w", err)
		return
	}
	if err := app.Run(); err != nil {
		log.Fatalf("failed to run app: %w", err)
		return
	}
}
