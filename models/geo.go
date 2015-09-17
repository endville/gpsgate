package models

import (
// "labix.org/v2/mgo/bson"
)

type Geo struct {
	Id          string   `bson:"_id"`
	GroupId     int64    `bson:"gid"`
	UserId      int64    `bson:"uid"`
	TerminalId  int64    `bson:"tid"`
	Location    Location `bson:"loc"`
	TimeStamp   int64    `bson:"ts"`
	Direction   float32  `bson:"direct"`
	Status      int      `bson:"status"`
	CellId      string   `bson:"cell"`
	Speed       float32  `bson:"s"` // 速度
	Info        string   `bson:"i"` // 三个数字：卫星数量、信号强度、电量
	Voltage     float32  `bson:"v"` // 电压
	Temperature int      `bson:"t"` // 温度
}

type Location struct {
	Longitude float32 `bson:"lng"`
	Latitude  float32 `bson:"lat"`
}
