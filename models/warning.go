package models

import (
	// "labix.org/v2/mgo/bson"
	"time"
)

const (
	WARNING_TYPE_NO_POWER         = iota + 1 // 断电
	WARNING_TYPE_LOW_VOLTAGE                 // 低电压
	WARNING_TYPE_HIGH_VOLTAGE                // 高电压
	WARNING_TYPE_MOVE                        // 位移
	WARNING_TYPE_OVER_SPEED                  // 超速
	WARNING_TYPE_VIBRATE                     // 震动
	WARNING_TYPE_HIGH_TEMPERATURE            // 高温
	WARNING_TYPE_LOW_TEMPERATURE             // 低温
	WARNING_TYPE_UNKONW                      // 未知
)

const (
	WARNING_STATE_NOT_SOLVED = iota + 1 // 未解决
	WARNING_STATE_SOLVED                // 已解决
)

const (
	WARNING_FLAG_NORMAL = iota + 1
)

type Warning struct {
	Id          int64   `json:"id"`
	TerminalSn  string  `json:"sn"`
	TerminalId  int64   `json:"tid"`
	UserId      int64   `json:"uid"`
	GroupId     int64   `json:"gid"`
	Longitude   float32 `json:"lng"`
	Latitude    float32 `json:"lat"`
	Speed       float32 `json:"speed"`
	Direction   float32 `json:"direction"`
	Status      int     `json:"status"`
	CellId      string  `json:"cellId"`
	Voltage     float32 `json:"voltage"`     // 电压
	Temperature int     `json:"temperature"` // 温度

	Type     int16     `json:"type"`  // 报警类型
	State    int16     `json:"state"` // 状态 初步设计为 未处理、已处理 2种
	Flag     int16     `json:"flag"`  // 标识
	CreateOn time.Time `json:"createOn" orm:"auto_now_add;type(datetime)"`
	ModifyOn time.Time `json:"modifyOn" orm:"auto_now;type(datetime)"`
}
