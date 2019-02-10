package main

import (
	"context"
	"log"
	"os"

	"github.com/evanphx/columbia/boundary"
	"github.com/evanphx/columbia/kernel"
	clog "github.com/evanphx/columbia/log"
	"github.com/evanphx/columbia/syscalls"
)

type closeProtect struct {
	io.Writer
}

func (_ closeProtect) Close() error {
	return nil
}

func main() {
	var wi boundary.WasmInterface
	wi.L = clog.L

	ctx := context.Background()

	kernel, err := kernel.NewKernel(wi.EnvModule())
	if err != nil {
		log.Fatal(err)
	}

	wi.Invoker = &syscalls.Invoker{
		Kernel: kernel,
	}

	cmd := "/bin/sh"
	args := []string{"sh", "-c", `echo "START: $(date)"`}
	// cmd := "/bin/signal"
	// args := []string{"signal"}

	proc, err := kernel.InitProcess(ctx, cmd, args, os.Environ())
	if err != nil {
		log.Fatal(err)
	}

	proc.HookupStdio(os.Stdin, closeProtect{os.Stdout}, closeProtect{os.Stderr})

	err = kernel.StartProcess(proc)
	if err != nil {
		log.Fatal(err)
	}
}
