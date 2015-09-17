package manager

import (
	"errors"
	"gpsgate/message"
	"gpsgate/service/session"
	"log"
	"time"
)

var manager *Manager = nil

type Manager struct {
	sessionManager *session.SessionManager
}

func (this *Manager) IsAlive(terminalSn string) bool {
	if _, ok := this.sessionManager.Get(terminalSn); ok {
		return true
	}
	return false
}

func (this *Manager) send(terminalSn string, msg message.ServerMessage) error {
	log.Printf("Try send message %s to %s\n", msg.String(), terminalSn)
	if sess, ok := this.sessionManager.Get(terminalSn); ok {
		//每次发送消息前清空反馈队列
		if (*sess).FeedbackChannel != nil {
			close((*sess).FeedbackChannel)
		}
		(*sess).FeedbackChannel = make(chan *message.TerminalMessage, session.FEEDBACK_CHANNEL_SIZE)
		if _, err := (*sess).Write([]byte(msg.String())); err != nil {
			return err
		}
	} else {
		return errors.New("Session " + terminalSn + " has disconnected.")
	}
	return nil
}

func (this *Manager) SendMessage(terminalSn string, message message.ServerMessage) error {
	err := this.send(terminalSn, message)
	if err != nil {
		return err
	}
	return nil
}

func (this *Manager) GetFeedback(terminalSn string, timeout time.Duration) (message.TerminalMessage, error) {
	if sess, ok := this.sessionManager.Get(terminalSn); ok {
		select {
		case feedback := <-(*sess).FeedbackChannel:
			return (*feedback), nil
		case <-time.After(timeout):
			return message.TerminalMessage{}, errors.New("Time out!")
		}
	} else {
		return message.TerminalMessage{}, errors.New("Session " + terminalSn + " has closed.")
	}
}

func New(sess *session.SessionManager) *Manager {
	if manager == nil {
		manager = &Manager{
			sessionManager: sess,
		}
	}
	return manager
}
