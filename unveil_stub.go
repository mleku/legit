//go:build !openbsd
// +build !openbsd

// Stub functions for GOOS that don't support unix.Unveil()

package main

func Unveil(_ string, _ string) error        { return nil }
func UnveilBlock() error                     { return nil }
func UnveilPaths(_ []string, _ string) error { return nil }
