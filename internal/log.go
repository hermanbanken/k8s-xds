package internal

import (
	"github.com/envoyproxy/go-control-plane/pkg/log"
	"go.uber.org/zap"
)

func xdsLog() log.Logger {
	std := zap.NewStdLog(zap.L())
	return log.LoggerFuncs{
		DebugFunc: func(s string, i ...interface{}) {
			std.Printf(s, i...)
		},
		InfoFunc: func(s string, i ...interface{}) {
			std.Printf(s, i...)
		},
		WarnFunc: func(s string, i ...interface{}) {
			std.Printf(s, i...)
		},
		ErrorFunc: func(s string, i ...interface{}) {
			std.Printf(s, i...)
		},
	}
}
