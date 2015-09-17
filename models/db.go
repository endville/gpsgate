package models

import (
	"errors"
	"fmt"
	"github.com/Unknwon/goconfig"
	"gopkg.in/mgo.v2"
	"log"
	"time"
)

var (
	MAX_DB_SESSION_NUM = 300
)

const (
	TIME_OUT = 10 * time.Second
)

var sess *mgo.Session = nil
var sessPool chan bool

type DBSession struct {
	*mgo.Session
}

func GetDBSession() (*DBSession, error) {
	if sess == nil {
		var err error
		sess, err = mgo.Dial("127.0.0.1")
		if err != nil {
			return nil, err
		}
	}

	select {
	case sessPool <- true:
		session := &DBSession{
			sess.New(),
		}
		return session, nil
	case <-time.After(TIME_OUT):
		errmsg := fmt.Sprintf("数据库连接数超过最大值(%d)", MAX_DB_SESSION_NUM)
		return nil, errors.New(errmsg)
	}

}

func (this *DBSession) Close() {
	this.Session.Close()
	<-sessPool
}

func init() {
	if cfg, err := goconfig.LoadConfigFile("config.ini"); err == nil {
		if value, err := cfg.Int(goconfig.DEFAULT_SECTION, "db_pool_size"); err == nil {
			MAX_DB_SESSION_NUM = value
		}
	} else {
		log.Println("读取配置文件失败[conf.ini]")
	}

	sessPool = make(chan bool, MAX_DB_SESSION_NUM)
	var err error
	sess, err = mgo.Dial("127.0.0.1")
	if err != nil {
		log.Println(err.Error())
	}
}
