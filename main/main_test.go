package main

import (
	"fmt"
	"github.com/docker/docker/pkg/reexec"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"syscall"
	"testing"
)

func stop() error {
	//return time.Sleep(100)
	return nil
}

func TestMain_interrupt(t *testing.T) {
	//process *os.Process := os.Process.Signal()

	cmd := &exec.Cmd{
		Path: reexec.Self(),
		Args: []string{"p2p-node"},
	}

	fmt.Println(cmd.Path)
	assert.PanicsWithValue(t, "boom", func() {
		go interrupt(stop)
	})

	os.Getegid()
	cmd.Process.Signal(syscall.SIGTERM)
}
