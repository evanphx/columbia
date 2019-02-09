package boundary

import (
	"context"
	"fmt"
)

func (w *WasmInterface) debug(ctx context.Context, arg int32) {
	fmt.Printf("debug: %d\n", arg)
}
