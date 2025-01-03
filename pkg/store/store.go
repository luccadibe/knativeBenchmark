package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Store struct {
	LogDirPath string `yaml:"logDirPath"`
}

func GetLogFilePath(logDirPath string) string {
	now := time.Now().Format("2006-01-02_15-04-05")
	return filepath.Join(logDirPath, fmt.Sprintf("%s.log", now))
}

func GetLogFileWriter(logDirPath string) *os.File {
	filePath := GetLogFilePath(logDirPath)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file: %v", err))
	}
	return file
}
