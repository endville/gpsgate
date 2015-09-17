package manager

import (
	"errors"
	"gpsgate/message"
	"gpsgate/service/session"
	"time"
)

const (
	RPC_SIGNAL_TEST = iota + 1
	RPC_SIGNAL_TEST_REPLY
)

var (
	terminalManager *Manager
	sessionManager  *session.SessionManager
)

type RPCSendMessageModel struct {
	TerminalSn   string
	Message      message.ServerMessage
	NeedFeedback bool
	Timeout      time.Duration
}

type RPCSessionModel struct {
	TerminalSn string    // 终端Sn号
	TerminalId int64     // 终端在数据库中的ID,方便查询数据库
	UserId     int64     // 终端所属用户的ID,方便查询数据库
	GroupId    int64     // 终端所属车队的ID,方便查询数据库
	ConnectOn  time.Time // 终端开始连接时间
	RemoteAddr string    // Session ID
}

type TerminalDelegate int

func (this *TerminalDelegate) SendMessage(model RPCSendMessageModel, result *message.TerminalMessage) error {
	err := terminalManager.SendMessage(model.TerminalSn, model.Message)
	if err != nil {
		*result = message.TerminalMessage{}
		return err
	}
	if model.NeedFeedback {
		if feedback, err := terminalManager.GetFeedback(model.TerminalSn, model.Timeout); err != nil {
			return err
		} else {
			*result = feedback
		}
	}
	return nil
}

func (this *TerminalDelegate) GetOnlineTerminalCount(param int, count *int) error {
	*count = sessionManager.Length()
	return nil
}

func (this *TerminalDelegate) GetOnlineTerminalSession(terminalSn string, session *RPCSessionModel) error {
	if sess, ok := sessionManager.Get(terminalSn); ok {
		(*session) = RPCSessionModel{
			TerminalSn: sess.TerminalSn,
			TerminalId: sess.TerminalId,
			UserId:     sess.UserId,
			GroupId:    sess.GroupId,
			ConnectOn:  sess.ConnectOn,
			RemoteAddr: sess.RemoteAddr,
		}
	} else {
		return errors.New("Terminal session not found.")
	}
	return nil
}

func (this *TerminalDelegate) GetOnlineTerminalSessions(param int, sessions *[]RPCSessionModel) error {
	sesses := sessionManager.Sessions()
	sessionList := make([]RPCSessionModel, len(sesses))
	i := 0
	for _, sess := range sesses {
		sessionList[i] = RPCSessionModel{
			TerminalSn: sess.TerminalSn,
			TerminalId: sess.TerminalId,
			UserId:     sess.UserId,
			GroupId:    sess.GroupId,
			ConnectOn:  sess.ConnectOn,
			RemoteAddr: sess.RemoteAddr,
		}
		i++
	}
	(*sessions) = sessionList
	return nil
}

func (this *TerminalDelegate) IsAlive(terminalSn string, result *int) error {
	if terminalManager.IsAlive(terminalSn) {
		*result = 1
	} else {
		*result = 0
	}
	return nil
}

func (this *TerminalDelegate) Kick(terminalSn string, result *int) error {
	if sessionManager.Delete(terminalSn) {
		*result = 1
	} else {
		*result = 0
	}
	return nil
}

func (this *TerminalDelegate) Test(signal int, reply *int) error {
	if signal != RPC_SIGNAL_TEST {
		return errors.New("Test error.")
	}
	*reply = RPC_SIGNAL_TEST_REPLY
	return nil
}

func init() {
	sessionManager = session.New()
	terminalManager = New(sessionManager)
}
