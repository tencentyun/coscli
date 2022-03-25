package logger

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
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
		logpath,
		rotatelogs.WithMaxAge(time.Duration(168)*time.Hour),
		rotatelogs.WithRotationTime(time.Duration(24)*time.Hour),
	)
	if err != nil {
		panic(err)
	}

	multiWriter := io.MultiWriter(fsWriter, os.Stdout)
	log.SetOutput(multiWriter)
	log.SetLevel(log.InfoLevel)
	forceColors := true
	if runtime.GOOS == "windows" {
		forceColors = false
	}
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:     forceColors,
		TimestampFormat: "2006-01-02 15:04:05", //时间格式
		FullTimestamp:   true,
	})
}
