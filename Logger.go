package binance_modules

import (
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Logger struct {
	*log.Logger
}

var loggerLock = sync.Mutex{}
var loggerInstance *Logger

func GetLogger() (*Logger, error) {
	if loggerInstance == nil {
		loggerLock.Lock()
		defer loggerLock.Unlock()
		if loggerInstance == nil {
			lg, err := newLogger("BM_logs.log")
			if err != nil {
				return loggerInstance, err
			}
			loggerInstance = lg
		}
	}
	return loggerInstance, nil
}

func newLogger(filename string) (*Logger, error) {
	lg := new(Logger)
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return lg, err
	}
	lg.Logger = log.New()
	lg.Out = file

	return lg, nil
}
