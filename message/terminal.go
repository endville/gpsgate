package message

import (
	"errors"
	"fmt"
	"strings"
)

type TerminalMessage struct {
	DateTime     string
	TerminalType string
	Version      string
	TerminalSn   string
	MessageType  string
	Params       []string
}

func (this *TerminalMessage) String() string {
	if len(this.Params) == 0 {
		return fmt.Sprintf("[%s,%s,%s,%s,%s]", this.DateTime, this.MessageType, this.Version, this.TerminalSn, this.MessageType)
	}
	return fmt.Sprintf("[%s,%s,%s,%s,%s,%s]", this.DateTime, this.TerminalType, this.Version, this.TerminalSn, this.MessageType, strings.Join(this.Params, ","))
}

func (this *TerminalMessage) Parse(message string) error {
	msgs, ok := SplitMessage(message)
	if ok && len(msgs) > 0 {
		message = msgs[0]
	}

	message = message[1 : len(message)-1] //去掉前后中括号
	params := strings.Split(message, ",")
	if len(params) < 2 {
		return errors.New("参数不足")
	}

	this.DateTime = params[0]
	this.TerminalType = params[1]
	this.Version = params[2]
	this.TerminalSn = params[3]
	this.MessageType = params[4]

	if len(params) > 5 {
		this.Params = params[5:]
	} else {
		this.Params = make([]string, 0)
	}

	return nil
}
