package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func stop() error {
	// return time.Sleep(100)
	return nil
}

func TestMain_interrupt(t *testing.T) {
	// process *os.Process := os.Process.Signal()
	file := filepath.Join(os.TempDir(), "config_test_datadir")
	defer os.Remove(file)

	cmd := &exec.Cmd{
		Path: os.Args[0],
		Args: []string{"--datadir " + file},
	}

	fmt.Println(cmd.Path)
	err := cmd.Start()
	assert.NoError(t, err)
	defer func() {
		if t.Failed() {
			cmd.Process.Kill()
		}
	}()
	// stdout, _ := cmd.StdoutPipe()
	// stderr, _ := cmd.StderrPipe()

	assert.PanicsWithValue(t, "boom", func() {
		go interrupt(stop)
	})

	os.Getegid()
	cmd.Process.Signal(syscall.SIGTERM)
}
