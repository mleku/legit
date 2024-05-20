package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"mleku.dev/git/slog"
	"mleku.net/legit/config"
	"mleku.net/legit/routes"
)

var log, chk = slog.New(os.Stderr)

func main() {
	slog.SetLogLevel(slog.Trace)
	var cfg string
	flag.StringVar(&cfg, "config", "./config.yaml", "path to config file")
	flag.Parse()

	c, err := config.Read(cfg)
	if err != nil {
		log.F.Ln(err)
	}

	if err := UnveilPaths([]string{
		c.Dirs.Static,
		c.Repo.ScanPath,
		c.Dirs.Templates,
	},
		"r"); err != nil {
		log.F.F("unveil: %s", err)
	}

	mux := routes.Handlers(c)
	addr := fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
	log.I.Ln("starting server on", addr)
	log.F.Ln(http.ListenAndServe(addr, mux))
}
