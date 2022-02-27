package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var logName = "coscli.log"

func init() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		dir = "."
	}
	logpath := dir + string(os.PathSeparator) + logName
	fsWriter, err := rotatelogs.New(
		logpath+"_%Y-%m-%d.log",
		rotatelogs.WithMaxAge(time.Duration(168)*time.Hour),
		rotatelogs.WithRotationTime(time.Duration(24)*time.Hour),
	)
	if err != nil {
		panic(err)
	}

	multiWriter := io.MultiWriter(fsWriter, os.Stdout)
	// log.SetReportCaller(true)
	log.SetOutput(multiWriter)
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		TimestampFormat: "2006-01-02 15:04:05", //时间格式
		FullTimestamp:   true,
	})

}
