package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/mleku/legit/config"
	"github.com/mleku/legit/routes"
	"github.com/mleku/lol"
)

var log, chk = lol.New(os.Stderr)

func main() {
	lol.SetLogLevel(lol.Trace)
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
