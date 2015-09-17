package message

import (
	"errors"
	"fmt"
	"strings"
)

type ServerMessage struct {
	DateTime    string
	MessageType string
	Params      []string
}

func (this *ServerMessage) String() string {
	if len(this.Params) == 0 {
		return fmt.Sprintf("[%s,%s]", this.DateTime, this.MessageType)
	}
	return fmt.Sprintf("[%s,%s,%s]", this.DateTime, this.MessageType, strings.Join(this.Params, ","))
}

func (this *ServerMessage) Parse(message string) error {
	msgs, ok := SplitMessage(message)
	if ok && len(msgs) > 0 {
		message = msgs[0]
	}

	message = message[1 : len(message)-1] //去掉前后中括号
	params := strings.Split(message, ",")
	if len(params) < 2 {
		return errors.New("解析服务器消息时发现参数不足")
	}

	this.DateTime = params[0]
	this.MessageType = params[1]

	if len(params) > 2 {
		this.Params = params[2:]
	} else {
		this.Params = make([]string, 0)
	}

	return nil
}
