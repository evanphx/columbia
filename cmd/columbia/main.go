package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/evanphx/columbia/boundary"
	"github.com/evanphx/columbia/kernel"
	clog "github.com/evanphx/columbia/log"
	"github.com/evanphx/columbia/syscalls"
	"github.com/spf13/pflag"
)

type closeProtect struct {
	*os.File
}

func (_ closeProtect) Close() error {
	return nil
}

var (
	fRoot = pflag.StringP("root", "r", "", "directory to mount as the root")
)

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

	pflag.Parse()

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

	inputArgs := pflag.Args()

	cmd := inputArgs[0]

	args := append([]string{filepath.Base(cmd)}, inputArgs[1:]...)

	proc, err := kernel.InitProcess(ctx, cmd, args, os.Environ(), *fRoot)
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
