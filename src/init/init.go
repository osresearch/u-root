// Copyright 2012 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// assumptions
// we've been booted into a ramfs with all this stuff unpacked and ready.
// we don't need a loop device mount because it's all there.
// So we run /go/bin/go build installcommand
// and then exec /buildbin/sh

package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"syscall"
)

type mount struct {
	source string
	target string
	fstype string
	flags  uintptr
	opts   string
	mode   os.FileMode
}

var (
	env = map[string]string{
		"PATH":            "/go/bin:/bin:/buildbin:/usr/local/bin:",
		"LD_LIBRARY_PATH": "/usr/local/lib",
		"GOROOT":          "/go",
		"GOPATH":          "/",
		"CGO_ENABLED":     "0",
	}

	namespace = []mount{
		{source: "proc", target: "/proc", fstype: "proc", flags: syscall.MS_MGC_VAL | syscall.MS_RDONLY, opts: "", mode: os.FileMode(0555)},
	}
)

func main() {
	log.Printf("Welcome to u-root")
	envs := []string{}
	for k, v := range env {
		os.Setenv(k, v)
		envs = append(envs, k+"="+v)
	}

	for _, m := range namespace {
		if err := os.MkdirAll(m.target, m.mode); err != nil {
			log.Printf("mkdir :%s: mode %o: %v\n", m.target, m.mode)
			continue
		}
		if err := syscall.Mount(m.source, m.target, m.fstype, m.flags, m.opts); err != nil {
			log.Printf("Mount :%s: on :%s: type :%s: flags %x: %v\n", m.source, m.target, m.fstype, m.flags, m.opts, err)
		}

	}
	// populate buildbin

	if commands, err := ioutil.ReadDir("/src"); err == nil {
		for _, v := range commands {
			name := v.Name()
			if name == "installcommand" || name == "init" {
				continue
			} else {
				destPath := path.Join("/buildbin", name)
				source := "/buildbin/installcommand"
				if err := os.Symlink(source, destPath); err != nil {
					log.Printf("Symlink %v -> %v failed; %v", source, destPath, err)
				}
			}
		}
	} else {
		log.Fatalf("Can't read %v; %v", "/src", err)
	}
	log.Printf("envs %v", envs)
	os.Setenv("GOBIN", "/buildbin")
	cmd := exec.Command("go", "install", "-x", "installcommand")
	installenvs := envs
	installenvs = append(envs, "GOBIN=/buildbin")
	cmd.Env = installenvs
	cmd.Dir = "/"

	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Printf("Run %v", cmd)
	err := cmd.Run()
	if err != nil {
		log.Printf("%v\n", err)
	}

	os.Setenv("GOBIN", "/bin")
	cmd = exec.Command("/buildbin/sh")
	envs = append(envs, "GOBIN=/bin")
	cmd.Env = envs
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Printf("Run %v", cmd)
	err = cmd.Run()
	if err != nil {
		log.Printf("%v\n", err)
	}
	log.Printf("init: /bin/sh returned!\n")
}