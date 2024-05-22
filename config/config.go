package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"mleku.dev/git/slog"
)

var log, chk = slog.New(os.Stderr)

type Config struct {
	Repo struct {
		ScanPath   string   `yaml:"scanPath"`
		Readme     []string `yaml:"readme"`
		MainBranch []string `yaml:"mainBranch"`
		Ignore     []string `yaml:"ignore,omitempty"`
	} `yaml:"repo"`
	Dirs struct {
		Templates string `yaml:"templates"`
		Static    string `yaml:"static"`
	} `yaml:"dirs"`
	Meta struct {
		Title       string `yaml:"title"`
		Description string `yaml:"description"`
	} `yaml:"meta"`
	Server struct {
		Name string `yaml:"name,omitempty"`
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`
}

func Read(f string) (c *Config, err error) {
	var b []byte
	if b, err = os.ReadFile(f); chk.E(err) {
		err = log.E.Err("reading config: %w", err)
		return
	}
	c = &Config{}
	if err = yaml.Unmarshal(b, c); chk.E(err) {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if c.Repo.ScanPath, err = filepath.Abs(c.Repo.ScanPath); err != nil {
		return nil, err
	}
	if c.Dirs.Templates, err = filepath.Abs(c.Dirs.Templates); err != nil {
		return nil, err
	}
	if c.Dirs.Static, err = filepath.Abs(c.Dirs.Static); err != nil {
		return nil, err
	}
	return
}
