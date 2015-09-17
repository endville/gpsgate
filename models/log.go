package models

const (
	LOG_LEVEL_DEBUG = 1 + iota
	LOG_LEVEL_INFO
	LOG_LEVEL_WORNING
	LOG_LEVEL_ERROR
)

type Log struct {
	ID       string `bson:"_id"`
	Level    int    `bson:"level"`
	Sn       string `bson:"sn"`
	Ip       string `bson:"ip"`
	Content  string `bson:"content"`
	Body     string `bson:"body"`
	InsertOn int64  `bson:"insert_on"`
}
