package message

import (
	"regexp"
)

var (
	messageRegex = regexp.MustCompile(`(\[.*?\])`) // 报文规则
)

type Messager interface {
	String() string
	Parse(message string) error
}

/*
将一段字符串分解成多个报文
*/
func SplitMessage(message string) ([]string, bool) {
	list := messageRegex.FindAllString(message, -1)
	if list != nil {
		return list, true
	}
	return nil, false
}
