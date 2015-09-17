package session

import (
	"errors"
	"github.com/Unknwon/goconfig"
	"gpsgate/message"
	"gpsgate/models/mysql"
	"log"
	"net"
	"sync"
	"time"
)

var (
	MAX_SESSION_NUM       = 20000 // 限制最大连接数
	FEEDBACK_CHANNEL_SIZE = 200   //
)

const (
	SESSION_STATE_CREATED   = iota + 1 // 刚刚创建
	SESSION_STATE_CONNECTED            // 被Manage管理
	SESSION_STATE_CLOSING              // 关闭中
	SESSION_STATE_CLOSED               // 已关闭
	SESSION_STATE_GONE                 // 和Manage脱离
)

// errors
var (
	ERROR_MAX_CLIENT         = errors.New("Error: MAX client.")
	ERROR_CONN_HAS_CLOSED    = errors.New("Error: Connection has closed.")
	ERROR_WRONG_TYPE         = errors.New("Error: Wrong type.")
	ERROR_NO_ACCESS_TERMINAL = errors.New("Error: Terminal not be accessed")
)

// 单例
var (
	sessionManager *SessionManager = nil
)

// Session 抽象
type Session struct {
	RemoteAddr        string    // Session ID
	TerminalSn        string    // 终端SN号
	TerminalId        int64     // 终端在数据库中的ID,方便查询数据库
	UserId            int64     // 终端所属用户的ID,方便查询数据库
	GroupId           int64     // 终端所属车队的ID,方便查询数据库
	ConnectOn         time.Time // 终端开始连接时间
	LastHandleMessage string    // 最后处理到的消息

	FeedbackChannel chan *message.TerminalMessage
	State           int

	options map[string]interface{}
	conn    *net.Conn
}

func NewSession(conn *net.Conn) *Session {
	return &Session{
		RemoteAddr: (*conn).RemoteAddr().String(),
		ConnectOn:  time.Now(),
		State:      SESSION_STATE_CREATED,
		conn:       conn,
		options:    make(map[string]interface{}),
	}
}

func (this *Session) Conn() *net.Conn {
	return this.conn
}

func (this *Session) Read(buffer []byte) (int, error) {
	if this.State < SESSION_STATE_CLOSING && this.conn != nil {
		return (*this.conn).Read(buffer)
	}
	return -1, errors.New("Can't read from a closing or closed session.")
}

func (this *Session) Write(buffer []byte) (int, error) {
	if this.State < SESSION_STATE_CLOSING && this.conn != nil {
		return (*this.conn).Write(buffer)
	}
	return -1, ERROR_CONN_HAS_CLOSED
}

func (this *Session) Close() {
	if this.conn != nil {
		this.State = SESSION_STATE_CLOSING
		(*this.conn).Close()
		this.conn = nil
		this.State = SESSION_STATE_CLOSED
		log.Printf("Session %s has closed. \n", this.TerminalSn)
	}
}

func (this *Session) Option(key string) (interface{}, error) {
	if value, ok := this.options[key]; ok {
		return value, nil
	}
	return nil, errors.New("Not found this option value")
}

func (this *Session) OptionInt(key string) (int, error) {
	if op, err := this.Option(key); err == nil {
		if val, ok := op.(int); ok {
			return val, nil
		} else {
			return 0, ERROR_WRONG_TYPE
		}
	} else {
		return 0, err
	}
}

func (this *Session) OptionInt32(key string) (int32, error) {
	if op, err := this.Option(key); err == nil {
		if val, ok := op.(int32); ok {
			return val, nil
		} else {
			return 0, ERROR_WRONG_TYPE
		}
	} else {
		return 0, err
	}
}

func (this *Session) OptionInt64(key string) (int64, error) {
	if op, err := this.Option(key); err == nil {
		if val, ok := op.(int64); ok {
			return val, nil
		} else {
			return 0, ERROR_WRONG_TYPE
		}
	} else {
		return 0, err
	}
}

func (this *Session) OptionFloat(key string) (float32, error) {
	if op, err := this.Option(key); err == nil {
		if val, ok := op.(float32); ok {
			return val, nil
		} else {
			return 0, ERROR_WRONG_TYPE
		}
	} else {
		return 0, err
	}
}

func (this *Session) OptionString(key string) (string, error) {
	if op, err := this.Option(key); err == nil {
		if val, ok := op.(string); ok {
			return val, nil
		} else {
			return "", ERROR_WRONG_TYPE
		}
	} else {
		return "", err
	}
}

func (this *Session) SetOption(key string, value interface{}) {
	this.options[key] = value
}

// Session 管理器
type SessionManager struct {
	mutex    sync.Mutex
	sessions map[string]*Session
}

// 将session进行管理，session的TerminalSn作为Key
func (this *SessionManager) Put(sess *Session) error {
	// 确保session都是合法的终端
	if sess.TerminalId == 0 || sess.TerminalSn == "" {
		return ERROR_NO_ACCESS_TERMINAL
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	if prevSess, ok := this.sessions[sess.TerminalSn]; ok {
		log.Printf("Remote address %s will be replaced by %s \n", prevSess.RemoteAddr, sess.RemoteAddr)
		prevSess.Close()
		(*prevSess).State = SESSION_STATE_GONE
	} else {
		if len(this.sessions) == MAX_SESSION_NUM {
			return ERROR_MAX_CLIENT
		}
		log.Printf("Put session: %s \n", sess.TerminalSn)
	}

	(*sess).State = SESSION_STATE_CONNECTED
	this.sessions[sess.TerminalSn] = sess
	o := mysql.GetOrm()

	// 设备上线
	if _, err := o.Raw("UPDATE terminal SET online_on = ? WHERE id = ?",
		time.Now(),
		sess.TerminalId,
	).Exec(); err != nil {
		log.Println("Update terminal online time failed:", err.Error())
	}
	return nil
}

func (this *SessionManager) Get(sn string) (*Session, bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if sess, ok := this.sessions[sn]; ok {
		return sess, true
	}
	return nil, false
}

func (this *SessionManager) Delete(sn string) bool {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if sess, ok := this.sessions[sn]; ok {
		delete(this.sessions, sn)
		sess.Close()
		(*sess).State = SESSION_STATE_GONE
		o := mysql.GetOrm()

		// 设备离线
		if _, err := o.Raw("UPDATE terminal SET offline_on = ? WHERE id = ?",
			time.Now(),
			sess.TerminalId,
		).Exec(); err != nil {
			log.Println("Update terminal offline time failed:", err.Error())
		}
		return true
	}
	return false
}

func (this *SessionManager) Length() int {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return len(this.sessions)
}

func (this *SessionManager) Sessions() map[string]*Session {
	return this.sessions
}

// 单例模式
func New() *SessionManager {
	if sessionManager == nil {
		sessionManager = &SessionManager{
			sessions: make(map[string]*Session, MAX_SESSION_NUM),
		}
	}
	return sessionManager
}

func init() {
	if cfg, err := goconfig.LoadConfigFile("config.ini"); err == nil {
		if value, err := cfg.Int(goconfig.DEFAULT_SECTION, "max_connection_number"); err == nil {
			MAX_SESSION_NUM = value
		}
	} else {
		log.Println("读取配置文件失败[conf.ini]")
	}
}
