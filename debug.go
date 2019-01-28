package columbia

import (
	"fmt"

	"github.com/go-interpreter/wagon/exec"
)

func (l *Loader) debug(p *exec.Process, arg int32) {
	fmt.Printf("debug: %d\n", arg)
}
