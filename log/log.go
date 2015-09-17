package log

import (
	"gpsgate/models"
	"labix.org/v2/mgo/bson"
	"time"
)

func Info(sn, ip, content, body string) {
	log(models.LOG_LEVEL_INFO, sn, ip, content, body)
}

func Debug(sn, ip, content, body string) {
	log(models.LOG_LEVEL_DEBUG, sn, ip, content, body)
}

func Worning(sn, ip, content, body string) {
	log(models.LOG_LEVEL_WORNING, sn, ip, content, body)
}

func Error(sn, ip, content, body string) {
	log(models.LOG_LEVEL_ERROR, sn, ip, content, body)
}

func log(level int, sn, ip, content, body string) {
	models.InsertDocument("endville-gps-log", "log", models.Log{
		ID:       bson.NewObjectId().Hex(),
		Sn:       sn,
		Ip:       ip,
		Level:    level,
		Content:  content,
		Body:     body,
		InsertOn: time.Now().Unix(),
	})
}
