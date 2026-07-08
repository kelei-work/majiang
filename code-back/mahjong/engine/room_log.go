package engine

import (
	"fmt"
	"log"
	"os"
	"time"

	"combine.com/utils/logger"
)

func (this *Room) ConfigLog() {
	roomid := this.GetRoomID()
	var err error
	now := time.Now()
	filename := fmt.Sprintf("log/%04d%02d%02d%02d%02d-%s.log", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), roomid)
	this.logFile, err = os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		logger.Errorf(err.Error())
		return
	}
	this.roomLogger = log.New(this.logFile, "", log.Lmicroseconds)
}

func (this *Room) Log(text string) {
	this.roomLogger.Println(text)
}

func (this *Room) CloseLogFile() {
	this.logFile.Close()
}
