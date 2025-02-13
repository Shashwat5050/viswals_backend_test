package logger

import (
	"io"

	"go.uber.org/zap"
)

// currently using default logger from zap.
func Init(out io.Writer, isDevEnv bool) (*zap.Logger, error) {
	if isDevEnv {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}
