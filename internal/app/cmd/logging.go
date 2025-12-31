package cmd

import (
	"log"
	"os"
)

func setupLogging(path string) (func(), error) {
	if path == "" {
		return func() {}, nil
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	log.SetOutput(file)
	return func() {
		_ = file.Close()
	}, nil
}
