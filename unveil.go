//go:build openbsd
// +build openbsd

package main

import (
	"golang.org/x/sys/unix"
)

func Unveil(path string, perms string) error {
	log.E.F("unveil: \"%s\", %s", path, perms)
	return unix.Unveil(path, perms)
}

func UnveilBlock() error {
	log.E.F("unveil: block")
	return unix.UnveilBlock()
}

func UnveilPaths(paths []string, perms string) (err error) {
	for _, path := range paths {
		if err = Unveil(path, perms); chk.E(err) {
			return
		}
	}
	return UnveilBlock()
}
