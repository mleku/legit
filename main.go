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
	log.I.Ln("logging")
	var cfg string
	flag.StringVar(&cfg, "config", "./config.yaml", "path to config file")
	flag.Parse()
	var err error
	var c *config.Config
	if c, err = config.Read(cfg); chk.E(err) {
		return
	}
	if err = UnveilPaths([]string{
		c.Dirs.Static,
		c.Repo.ScanPath,
		c.Dirs.Templates,
	}, "r"); chk.E(err) {
		return
	}
	mux := routes.Handlers(c)
	addr := fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
	log.I.Ln("starting server on", addr)
	log.F.Ln(http.ListenAndServe(addr, mux))
}
