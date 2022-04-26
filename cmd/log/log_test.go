package log

import (
	"bufio"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	l := New()

	if l == nil {
		t.Error("creating new logger resulted in nil")
	}
}

func TestWriteToFile(t *testing.T) {
	testFilePath := strconv.Itoa(time.Now().Nanosecond())

	logFile, err := os.OpenFile(testFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0755)
	if err != nil {
		t.Errorf("creating test log file: %v", err)
	}
	defer os.Remove(testFilePath)

	l := New().SetOutput(logFile)

	l.LogWithFields(LevelInfo, Fields{
		"test_key":   "age",
		"test_value": 33,
	}, "just a test")

	readFrom, err := os.Open(testFilePath)
	if err != nil {
		t.Errorf("opening test log file for reading: %v", err)
	}
	defer readFrom.Close()

	scanner := bufio.NewScanner(readFrom)
	scanner.Scan()
	if scanner.Text() == "" {
		t.Error("test log file is empty")
	}
}
