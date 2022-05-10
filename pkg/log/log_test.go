package log

import "testing"

func TestLog(t *testing.T) {
	Debug("debug message")
	Debugf("debug message %s", "arg")
	DebugJ("debug message")

	Info("info message")
	Infof("info message %s", "arg")
	InfoJ("info message")

	Warn("warn message")
	Warnf("warn message %s", "arg")
	WarnJ("warn message")

	Error("error message")
	Errorf("error message %s", "arg")
	ErrorJ("error message")

	// Fatal("fatal message")
	// Fatalf("fatal message %s", "arg")
	// FatalJ("fatal message")

	Print("some message")
	Printf("some message %s", "arg")
}
