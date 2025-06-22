package logger

import (
	"log"
	"os"
)

type Logger struct {
	info    *log.Logger
	warning *log.Logger
	error_  *log.Logger
	active  bool
}

var FSLogger Logger

func InitializeLogger(active bool, logfilename string) {
	if active {

		file, err := os.OpenFile(logfilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal(err)
		}

		info := log.New(file, "FSForensics|INFO: ", log.Ldate|log.Ltime)
		warning := log.New(file, "FSForensics|WARNING: ", log.Ldate|log.Ltime)
		error_ := log.New(file, "FSForensics|ERROR: ", log.Ldate|log.Ltime)
		FSLogger = Logger{info: info, warning: warning, error_: error_, active: active}
	} else {
		FSLogger = Logger{active: active}
	}

}

func (logger Logger) Info(msg string) {
	if logger.active {
		logger.info.Println(msg)
	}
}

func (logger Logger) Error(msg any) {
	if logger.active {
		logger.error_.Println(msg)
	}
}

func (logger Logger) Warning(msg string) {
	if logger.active {
		logger.warning.Println(msg)
	}
}
