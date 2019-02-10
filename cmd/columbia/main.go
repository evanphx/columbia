package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/pprof"

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
	cpuprofile := os.Getenv("CPUPROFILE")
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		fmt.Printf("pprof: profiling started\n")
	}

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

	if cpuprofile != "" {
		pprof.StopCPUProfile()
		fmt.Printf("pprof: profiling finished\n")
	}

	if err != nil {
		log.Fatal(err)
	}
}
