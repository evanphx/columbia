package loader

import "github.com/evanphx/columbia/exec"

type Module struct {
	loader *Loader
	Module *exec.PreparedModule
}
