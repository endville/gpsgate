package message

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestParseTerminalMessage(t *testing.T) {
	msg := new(TerminalMessage)
	err := msg.Parse("[1000-12-15 10:00:00,1,V1.0.0,030600001,T3,1,E,113.252432,N,22.564152,50.6,270.5,1]")
	if err != nil {
		t.Error(err.Error())
	} else {
		if !(msg.DateTime == "1000-12-15 10:00:00" &&
			msg.TerminalType == "1" &&
			msg.Version == "V1.0.0" &&
			msg.TerminalNumber == "030600001" &&
			msg.MessageType == "T3" &&
			len(msg.Params) == 8) {
			t.Error("解析错误")
			b, _ := json.Marshal(&msg)
			fmt.Println(string(b))
		}
	}
}

func TestTerminalPrint(t *testing.T) {
	var msg TerminalMessage
	msg.DateTime = "1000-12-15 10:00:00"
	msg.TerminalType = "1"
	msg.Version = "V1.0.0"
	msg.TerminalNumber = "030600001"
	msg.MessageType = "T3"
	msg.Params = []string{"1", "E", "113.252432", "N", "22.564152", "50.6", "270.5", "1"}
	if msg.String() != "[1000-12-15 10:00:00,1,V1.0.0,030600001,T3,1,E,113.252432,N,22.564152,50.6,270.5,1]" {
		t.Error("转字符串错误")
	}
}
