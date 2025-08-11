package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

// broker simply execs the go4pack binary (api service) passing through args/env.
// This can be extended later to supervise, restart, or hot-reload.
func main() {
	// Determine path to the real binary (installed alongside or built in same dir)
	selfDir, err := os.Executable()
	if err != nil {
		log.Fatalf("broker: cannot resolve executable path: %v", err)
	}
	dir := filepathDir(selfDir)
	target := dir + "/go4pack"
	if _, err := os.Stat(target); err != nil {
		log.Fatalf("broker: target binary not found at %s: %v", target, err)
	}
	// Prepare to exec
	cmd := exec.Command(target, os.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		// If it exited with non-zero, propagate exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok2 := exitErr.Sys().(syscall.WaitStatus); ok2 {
				os.Exit(status.ExitStatus())
			}
		}
		log.Fatalf("broker: execution error: %v", err)
	}
}

// small helper avoiding extra import just for filepath
func filepathDir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}
