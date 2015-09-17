package message

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestParseServerMessage(t *testing.T) {
	msg := new(ServerMessage)
	err := msg.Parse("[2011-12-15 20:00:00,S1,1,2,3]")
	if err != nil {
		t.Error(err.Error())
	} else {
		if !(msg.DateTime == "2011-12-15 20:00:00" &&
			msg.MessageType == "S1" &&
			len(msg.Params) == 3) {
			t.Error("解析错误")
			b, _ := json.Marshal(&msg)
			fmt.Println(string(b))
		}
	}
}

func TestServerPrint(t *testing.T) {
	var msg ServerMessage
	msg.DateTime = "2011-12-15 20:00:00"
	msg.MessageType = "T3"
	msg.Params = []string{"1", "2", "3"}
	if msg.String() != "[2011-12-15 20:00:00,T3,1,2,3]" {
		fmt.Println(msg.String())
		t.Error("转字符串错误")
	}
}
