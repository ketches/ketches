package log

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"
)

var logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds)

type level string

type jlog struct {
	Timestamp string `json:"ts"`
	Level     level  `json:"level"`
	Position  string `json:"pos"`
	Text      string `json:"text"`
}

const (
	ldebug level = "DEBUG"
	linfo  level = "INFO"
	lwarn  level = "WARN"
	lerror level = "ERROR"
	lfatal level = "FATAL"
	lclean level = ""

	jtsf string = "2006/01/02 15:04:05.000000"
)

func Debug(args ...interface{}) {
	text := fmt.Sprint(args...)
	print(text, ldebug)
}

func Debugf(format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	print(text, ldebug)
}

func DebugJ(args ...interface{}) {
	text := fmt.Sprint(args...)
	j, _ := json.Marshal(jlog{Level: ldebug, Text: text, Timestamp: jtimestamp(), Position: jpos()})
	fmt.Println(string(j))
}

func Info(args ...interface{}) {
	text := fmt.Sprint(args...)
	print(text, linfo)
}

func Infof(format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	print(text, linfo)
}

func InfoJ(args ...interface{}) {
	text := fmt.Sprint(args...)
	j, _ := json.Marshal(jlog{Level: linfo, Text: text, Timestamp: jtimestamp(), Position: jpos()})
	fmt.Println(string(j))
}

func Warn(args ...interface{}) {
	text := fmt.Sprint(args...)
	print(text, lwarn)
}

func Warnf(format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	print(text, lwarn)
}

func WarnJ(args ...interface{}) {
	text := fmt.Sprint(args...)
	j, _ := json.Marshal(jlog{Level: lwarn, Text: text, Timestamp: jtimestamp(), Position: jpos()})
	fmt.Println(string(j))
}

func Error(args ...interface{}) {
	text := fmt.Sprint(args...)
	print(text, lerror)
}

func Errorf(format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	print(text, lerror)
}

func ErrorJ(args ...interface{}) {
	text := fmt.Sprint(args...)
	j, _ := json.Marshal(jlog{Level: lerror, Text: text, Timestamp: jtimestamp(), Position: jpos()})
	fmt.Println(string(j))
}

// Fatal log fatal message and exit.
func Fatal(args ...interface{}) {
	text := fmt.Sprint(args...)
	print(text, lfatal)
	os.Exit(1)
}

// Fatalf log format fatal message and exit.
func Fatalf(format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	print(text, lfatal)
	os.Exit(1)
}

// FatalJ log json fatal message and exit.
func FatalJ(args ...interface{}) {
	text := fmt.Sprint(args...)
	j, _ := json.Marshal(jlog{Level: lfatal, Text: text, Timestamp: jtimestamp(), Position: jpos()})
	fmt.Println(string(j))
	os.Exit(1)
}

func Print(args ...interface{}) {
	logger.Print(args...)
}

func Printf(format string, args ...interface{}) {
	logger.Printf(format, args...)
}

func print(text string, l level) {
	logger.Printf("[%s] %s", l, text)
}

func jtimestamp() string {
	return time.Now().Format(jtsf)
}

func jpos() string {
	_, f, l, _ := runtime.Caller(2)
	return fmt.Sprintf("%s:%d", f, l)
}
