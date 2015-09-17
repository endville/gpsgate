package manager

import (
	"fmt"
	"gpsgate/message"
	"net/rpc"
	"testing"
	"time"
)

func TestGetInfo(t *testing.T) {
	client, err := rpc.DialHTTP("tcp", "127.0.0.1:8502")
	if err != nil {
		fmt.Println("链接rpc服务器失败:", err)
	}
	var reply int
	err = client.Call("TerminalDelegate.SendMessage", RPCSendMessageModel{
		"'test'", message.ServerMessage{
			time.Now().Format("2006-01-02 15:04:05"),
			"T1",
			[]string{"1", "2"},
		}}, &reply)
	if err != nil {
		fmt.Println("调用远程服务失败", err)
	}
	fmt.Println("远程服务返回结果：", reply)
	if reply != 1 {
		t.Error("reply should be 1.")
	}
}
