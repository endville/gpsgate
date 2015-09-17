package service

import (
	"fmt"
	"gpsgate/lib"
	_log "gpsgate/log"
	"gpsgate/message"
	"gpsgate/service/handlers"
	"gpsgate/service/manager"
	"gpsgate/service/session"
	// "labix.org/v2/mgo/bson"
	"log"
	"net"
	"net/http"
	"net/rpc"
	// "strings"
	"time"
)

var (
	timeWheel      *lib.TimingWheel
	sessionManager *session.SessionManager
)

const (
	KILL_TIME_OUT = 6 * time.Minute
)

// 监听GPS设备上发消息
func StartTerminalListeningService(port int) {
	defer func() {
		if err := recover(); err != nil {
			_log.Error("Unknonw", "Unknonw", fmt.Sprintf("Listener Error: %v\n", err), "")
		}
	}()
	// 绑定端口
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Println("Has error: %s", err.Error())
		return
	}

	// 监听循环
	for {
		// 接受客户端链接
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Has error when accepting terminal message: %s", err.Error())
			continue
		}
		// 分发处理客户端链接
		go distributeTerminalConnection(&conn, timeWheel, KILL_TIME_OUT)
	}
}

// 监听来自管理端的消息
// func StartTerminalManageService(port int) {
// 	// 绑定端口
// 	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
// 	if err != nil {
// 		log.Println("Has error: %s", err.Error())
// 		return
// 	}

// 	// 监听循环
// 	for {
// 		// 接受客户端链接
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			log.Println("Has error when accepting manage message: %s", err.Error())
// 			continue
// 		}
// 		// 分发处理管理消息
// 		go distributeManageSignal(&conn)
// 	}
// }

// 监听来自管理端的消息(RPC)
func StartRPCTerminalManageService(port int) {
	delegate := new(manager.TerminalDelegate)
	rpc.Register(delegate)
	rpc.HandleHTTP()
	// 绑定端口
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Println("Has error: %s", err.Error())
		return
	}
	http.Serve(listener, nil)
}

// 分发处理管理消息
// func distributeManageSignal(conn *net.Conn) {
// 	defer func() {
// 		(*conn).Close()
// 	}()

// 	// 客户端
// 	log.Printf("Manage: %s\n", (*conn).RemoteAddr())
// 	// 数据缓冲区大小
// 	databuf := make([]byte, 1024)
// 	n, err := (*conn).Read(databuf)
// 	if err != nil {
// 		log.Println(err.Error())
// 		return
// 	}

// 	data := databuf[:n]
// 	messages := message.SplitMessage(string(data))

// 	var msgPart []string
// 	var lenOfMsgPart int
// 	var sessionID string
// 	var messageType string
// 	var params []string
// 	for _, msg := range messages {
// 		//msg format: [sessionID,messageType,params...][...]
// 		msg = msg[1 : len(msg)-1]
// 		msgPart = strings.Split(msg, ",")
// 		lenOfMsgPart = len(msgPart)
// 		if lenOfMsgPart >= 2 {
// 			sessionID = msgPart[0]
// 			messageType = msgPart[1]
// 			if lenOfMsgPart > 2 {
// 				params = msgPart[2:]
// 			} else {
// 				params = []string{}
// 			}
// 		} else {
// 			log.Printf("解析[%s]时发现参数不足.\r\n", msg)
// 			continue
// 		}

// 		terminalManager.SendMessage(sessionID, message.ServerMessage{
// 			time.Now().Format("2006-01-02 15:04:05"),
// 			messageType,
// 			params,
// 		})
// 	}
// }

// 分发处理客户端链接
func distributeTerminalConnection(conn *net.Conn, tw *lib.TimingWheel, timeout time.Duration) {
	var currSession *session.Session

	defer func() {
		if err := recover(); err != nil {
			_log.Error(currSession.TerminalSn, currSession.RemoteAddr, fmt.Sprintf("%v", err), currSession.LastHandleMessage)
		}
		if currSession.State == session.SESSION_STATE_CONNECTED ||
			currSession.State == session.SESSION_STATE_CLOSING ||
			currSession.State == session.SESSION_STATE_CLOSED {
			sessionManager.Delete(currSession.TerminalSn)
		} else {
			currSession.Close()
		}

		log.Println("Goroutine is over. Close terminal connect.")
	}()
	currSession = session.NewSession(conn)
	// 客户端
	log.Printf("Client: %s\n", currSession.RemoteAddr)

	// 数据缓冲区大小
	databuf := make([]byte, 1024)

	var chBuf chan int = make(chan int, 1)
	var chErr chan error = make(chan error, 1)
	// 保持连接并数据循环
	for {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					_log.Error(currSession.TerminalSn, currSession.RemoteAddr, fmt.Sprintf("%v", err), currSession.LastHandleMessage)
					chErr <- fmt.Errorf("严重错误：%v", err)
				}
			}()
			// 读取数据
			n, err := currSession.Read(databuf)
			if err != nil {
				chErr <- err
			}
			chBuf <- n
		}()
		if func() bool {
			select {
			case bufLen := <-chBuf:
				if messageList, ok := message.SplitMessage(string(databuf[:bufLen])); ok {
					for _, msg := range messageList {
						log.Println("Recieve:", msg)
						parseMsg := new(message.TerminalMessage)
						if err := parseMsg.Parse(msg); err == nil {
							return handlers.HandleMessage(currSession, parseMsg)
						} else {
							log.Println(err.Error())
						}
					}
				}
			case err := <-chErr:
				log.Println(err.Error())
				return false
			case <-tw.After(timeout):
				log.Printf("Socket timeout.")
				return false
			}
			return true
		}() {
			continue
		} else {
			break
		}
	}
	close(chBuf)
	close(chErr)
	chBuf = nil
	chErr = nil
	databuf = nil
}

func init() {
	sessionManager = session.New()
	timeWheel = lib.NewTimingWheel(1*time.Second, 600) //设最大超时时间
}
