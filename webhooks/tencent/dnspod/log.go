// SPDX-License-Identifier: MIT

package dnspod

import (
	"log/slog"
	"os"
)

func newLogger() (*slog.Logger, *slog.LevelVar) {
	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	return log, logLevel
}

func (s *Solver) SetLogLevel(level string) error {
	if err := s.logLevel.UnmarshalText([]byte(level)); err != nil {
		s.logErr(err, "failed to parse log level, valid values are: debug, info, warn, error")
		return err
	}
	return nil
}

// logErr logs err at Error level together with msg and optional key-value pairs.
// Parameter order (err, msg, args...) matches the call-site convention used
// throughout this package; it differs from slog's (msg, args...) signature.
func (s *Solver) logErr(err error, msg string, args ...any) {
	args = append(args, "error", err)
	s.log.Error(msg, args...)
}
