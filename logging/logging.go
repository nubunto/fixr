package logging

import (
	"io"
	"log"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
)

func Start(writeTo io.Writer) {
	logType := log.Ldate | log.Ltime | log.Lshortfile
	infoLogger = log.New(writeTo, "INFO: ", logType)
	errorLogger = log.New(writeTo, "ERROR :", logType)
}

func Error(err error) {
	errorLogger.Println(err)
}

func Info(msg string) {
	infoLogger.Println(msg)
}
