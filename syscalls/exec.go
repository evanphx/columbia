package syscalls

import (
	"context"

	"github.com/evanphx/columbia/abi"
	"github.com/evanphx/columbia/fs"
	"github.com/evanphx/columbia/kernel"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"
)

func copyStringArray(p *kernel.Task, addr int32) ([]string, error) {
	var args []string

	ptr := addr
	for {
		var addr int32
		err := p.CopyIn(ptr, &addr)
		if err != nil {
			return nil, err
		}

		if addr == 0 {
			break
		}

		str, err := p.ReadCString(addr)
		if err != nil {
			return nil, err
		}

		args = append(args, string(str))

		ptr += 4
	}

	return args, nil
}

func sysExecve(ctx context.Context, l hclog.Logger, task *kernel.Task, args SysArgs) int32 {
	var (
		pathAddr = args.Args.R0
		argvAddr = args.Args.R1
		envpAddr = args.Args.R2
	)

	path, err := task.ReadCString(pathAddr)
	if err != nil {
		l.Error("error reading path addr", "error", err)
		return -abi.ENOSYS
	}

	execArgs, err := copyStringArray(task, argvAddr)
	if err != nil {
		l.Error("error copying argv data", "error", err)
		return -abi.ENOSYS
	}

	execEnv, err := copyStringArray(task, envpAddr)
	if err != nil {
		l.Error("error copying argv data", "error", err)
		return -abi.ENOSYS
	}

	_, err = task.Process.Kernel.SetupProcess(ctx, task.Process, string(path), execArgs, execEnv)
	if err != nil {
		if errors.Cause(err) == fs.ErrUnknownPath {
			return -abi.ENOENT
		}

		l.Error("unable to exec process", "error", err, "path", path)
		return -abi.ENOEXEC
	}

	go task.Process.Kernel.StartProcess(task.Process)

	return 0
}

func sysExitGroup(ctx context.Context, l hclog.Logger, task *kernel.Task, args SysArgs) int32 {
	task.Exit(int(args.Args.R0))
	return 0
}

func init() {
	Syscalls[11] = sysExecve
	Syscalls[1] = sysExitGroup
	Syscalls[252] = sysExitGroup
}
