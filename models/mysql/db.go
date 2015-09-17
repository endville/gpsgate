package mysql

import (
	"errors"
	"fmt"
	"github.com/Unknwon/goconfig"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	"gpsgate/models"
	"log"
	"net/url"
	"time"
)

var (
	dbUser     = "root"
	dbPassword = "endville!"
	dbHost     = "127.0.0.1"
	dbPort     = "3306"
	dbName     = "endville_gps"
)

var (
	ERROR_NOT_FOUND = errors.New("未找到匹配记录")
)

func GetOrm() orm.Ormer {
	o := orm.NewOrm()
	o.Using("default")
	return o
}

func AddWarning(warning *models.Warning) (int64, error) {
	o := GetOrm()
	if res, err := o.Raw("INSERT warning(terminal_sn,terminal_id,user_id,group_id,longitude,latitude,speed,direction,status,cell_id,voltage,temperature,type,state,flag,create_on,modify_on) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		warning.TerminalSn,
		warning.TerminalId,
		warning.UserId,
		warning.GroupId,
		warning.Longitude,
		warning.Latitude,
		warning.Speed,
		warning.Direction,
		warning.Status,
		warning.CellId,
		warning.Voltage,
		warning.Temperature,
		warning.Type,
		warning.State,
		warning.Flag,
		time.Now().Format("2006-01-02 15:04:05"),
		time.Now().Format("2006-01-02 15:04:05"),
	).Exec(); err == nil {
		insertId, _ := res.LastInsertId()
		return insertId, nil
	} else {
		return 0, err
	}
}

func init() {
	if cfg, err := goconfig.LoadConfigFile("config.ini"); err == nil {
		dbUser, _ = cfg.GetValue(goconfig.DEFAULT_SECTION, "dbuser")
		dbPassword, _ = cfg.GetValue(goconfig.DEFAULT_SECTION, "dbpass")
		dbHost, _ = cfg.GetValue(goconfig.DEFAULT_SECTION, "dbhost")
		dbPort, _ = cfg.GetValue(goconfig.DEFAULT_SECTION, "dbport")
		dbName, _ = cfg.GetValue(goconfig.DEFAULT_SECTION, "dbname")
	} else {
		log.Println("读取配置文件conf.ini失败")
	}

	orm.DefaultTimeLoc = time.Local
	maxIdle := 50  //(可选)  设置最大空闲连接
	maxConn := 100 //(可选)  设置最大数据库连接 (go >= 1.2)
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	) + "&loc=" + url.QueryEscape("Local")

	if err := orm.RegisterDataBase("default", "mysql",
		connStr,
		maxIdle,
		maxConn,
	); err != nil {
		log.Println(err.Error())
	}

}
