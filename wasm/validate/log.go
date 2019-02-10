// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package validate

import (
	"io/ioutil"
	"log"
	"os"
)

var PrintDebugInfo = false

func init() {
	if PrintDebugInfo {
		w := ioutil.Discard

		w = os.Stderr

		logger = log.New(w, "", log.Lshortfile)
		log.SetFlags(log.Lshortfile)
	} else {
		logger = noopLogger{}
	}
}

type LoggerInterface interface {
	Printf(str string, args ...interface{})
}

type noopLogger struct{}

func (_ noopLogger) Printf(str string, args ...interface{}) {}

var (
	realLogger *log.Logger
	logger     LoggerInterface
	logging    bool
)
