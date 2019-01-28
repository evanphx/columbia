package exec

import "fmt"

var showDebug = false

func Debugf(str string, args ...interface{}) {
	if !showDebug {
		return
	}

	fmt.Printf(str, args...)
}
