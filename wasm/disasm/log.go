// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package disasm

import (
	"io/ioutil"
	"log"
	"os"
)

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

func SetDebugMode(l bool) {
	if l {
		w := ioutil.Discard
		logging = l

		w = os.Stderr

		realLogger = log.New(w, "", log.Lshortfile)
		realLogger.SetFlags(log.Lshortfile)
		logger = realLogger
	} else {
		logger = noopLogger{}
	}
}

func init() {
	SetDebugMode(false)
}
