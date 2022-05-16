package trace

import (
	"flag"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var levelFlag = zap.DebugLevel

func init() {
	flag.Var(&levelFlag, "loglevel", "set the loglevel")
}

func InstallZap() {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		os.Stderr,
		levelFlag,
	)
	z := zap.New(core)
	zap.ReplaceGlobals(z)
}
