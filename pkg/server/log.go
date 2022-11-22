package server

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
)

const (
	loggerDomainName = "dogesyncer"
)

// newFileLogger returns logger instance that writes all logs to a specified file.
//
// If log file can't be created, it returns an error
func newFileLogger(config *ServerConfig) (hclog.Logger, error) {
	logFileWriter, err := os.OpenFile(
		config.LogFilePath,
		os.O_CREATE+os.O_RDWR+os.O_APPEND,
		0640,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create log file, %w", err)
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:   loggerDomainName,
		Level:  config.LogLevel,
		Output: logFileWriter,
	}), nil
}

// newCLILogger returns minimal logger instance that sends all logs to standard output
func newCLILogger(config *ServerConfig) hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Name:  loggerDomainName,
		Level: config.LogLevel,
	})
}

// newLoggerFromConfig creates a new logger which logs to a specified file.
//
// If log file is not set it outputs to standard output ( console ).
// If log file is specified, and it can't be created the server command will error out
func newLoggerFromConfig(config *ServerConfig) (hclog.Logger, error) {
	if config.LogFilePath != "" {
		fileLoggerInstance, err := newFileLogger(config)
		if err != nil {
			return nil, err
		}

		return fileLoggerInstance, nil
	}

	return newCLILogger(config), nil
}
