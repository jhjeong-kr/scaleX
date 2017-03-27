package log

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	config "cperfc/config"
	log "github.com/Sirupsen/logrus"
)

func init() {
}

func Logging() {
	if config.LogFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		myTextFormatter := new(TextFormatter)
		myTextFormatter.TimestampFormat = "01-02 15:04:05.000"
		myTextFormatter.ShortTimestamp = false
		log.SetFormatter(myTextFormatter)
	}
	level, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		level = log.InfoLevel
	}
    log.SetOutput(os.Stderr)
	log.SetLevel(level)
}

func Fatal(args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Fatal(args...)
}

func Error(args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Error(args...)
}

func Warn(args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Warn(args...)
}

func Info(args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Info(args...)
}

func Debug(args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Debug(args...)
}

func Fatalf(format string, args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Fatalf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Errorf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Warnf(format, args...)
}

func Infof(format string, args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Infof(format, args...)
}

func Debugf(format string, args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Debugf(format, args...)
}

func Print(args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Info(args...)
}

func Println(args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Info(args...)
}

func Printf(format string, args ...interface{}) {
	prefix := findCaller()
	log.WithFields(log.Fields{"prefix": prefix}).Infof(format, args...)
}

func findCaller() string {
	for i := 2; ; i++ {
		pc, filepath, line, ok := runtime.Caller(i)
		if !ok {
			return "Unknown"
		}
		parts := strings.Split(filepath, "/")
		dir := parts[len(parts)-2]
		file := parts[len(parts)-1]
		if (dir != "log") || (file != "log.go") {
			parts = strings.Split(runtime.FuncForPC(pc).Name(), ".")
			return fmt.Sprintf("%s(%s:%d)", file, parts[len(parts) - 1], int(line))
		}
	}
	return "Unknown"
}
