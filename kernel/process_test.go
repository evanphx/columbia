package kernel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestWait(t *testing.T) {
	n := neko.Modern(t)

	n.It("detects another process has exitted", func(t *testing.T) {
		k, err := NewKernel(nil)
		require.NoError(t, err)

		parent := &Process{
			Kernel: k,
			Pid:    1,
			pg:     &ProcessGroup{},
		}

		parent.pg.PushBack(parent)

		child := &Process{
			Kernel: k,
			Pid:    2,
			pg:     parent.pg,
		}

		parent.pg.PushBack(child)

		child.Exit(1)

		ctx := context.Background()
		ctx, f := context.WithTimeout(ctx, 2*time.Second)
		defer f()

		pid, ret, err := parent.WaitAnyChild(ctx)
		require.NoError(t, err)

		require.Equal(t, 2, pid)

		require.Equal(t, 1, ret.Code)
	})

	n.It("waits for a child to exit", func(t *testing.T) {
		k, err := NewKernel(nil)
		require.NoError(t, err)

		parent := &Process{
			Kernel: k,
			Pid:    1,
			pg:     &ProcessGroup{},
		}

		parent.pg.PushBack(parent)

		child := &Process{
			Kernel: k,
			Pid:    2,
			pg:     parent.pg,
		}

		parent.pg.PushBack(child)

		go func() {
			time.Sleep(time.Second)
			child.Exit(1)
		}()

		ctx := context.Background()
		ctx, f := context.WithTimeout(ctx, 5*time.Second)
		defer f()

		pid, ret, err := parent.WaitAnyChild(ctx)
		require.NoError(t, err)

		require.Equal(t, 2, pid)

		require.Equal(t, 1, ret.Code)
	})

	n.Meow()
}
