package handlers

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	gopkg "gopkg.in/mgo.v2"
	"gpsgate/message"
	"gpsgate/models"
	"gpsgate/models/mysql"
	"gpsgate/service/session"
	"labix.org/v2/mgo/bson"
	"log"
	"strconv"
	"time"
)

var (
	sessionManager *session.SessionManager
)

//如果返回false会直接断开这个session
func HandleMessage(sess *session.Session, msg *message.TerminalMessage) bool {
	if msg == nil {
		return false
	} else {
		(*sess).LastHandleMessage = msg.String()
	}

	if msg.TerminalSn != "" {
		if sess.TerminalSn == "" {
			(*sess).TerminalSn = msg.TerminalSn
			var result []orm.Params
			o := mysql.GetOrm()
			num, err := o.Raw("SELECT id,user_id,group_id FROM terminal WHERE terminal_sn = ?", msg.TerminalSn).Values(&result)
			if err == nil && num > 0 {
				id, _ := strconv.ParseInt(result[0]["id"].(string), 10, 64)
				uid, _ := strconv.ParseInt(result[0]["user_id"].(string), 10, 64)
				gid, _ := strconv.ParseInt(result[0]["group_id"].(string), 10, 64)
				(*sess).TerminalId = id
				(*sess).UserId = uid
				(*sess).GroupId = gid
				if err := sessionManager.Put(sess); err != nil {
					log.Println(err.Error())
					return false
				}
			} else {
				// 新设备只允许登录？怎么处理再考虑考虑
				// 不是登录请求直接踢出
				if msg.MessageType != "T1" {
					sess.Close()
					log.Println("禁止未注册设备提交T1之外的消息，session被关闭")
					return false
				}
			}
		} else {
			if sess.TerminalSn != msg.TerminalSn {
				log.Println("TerminalSn 异常.")
				return false
			}
		}

		//分别处理各种消息
		switch msg.MessageType {
		case "T0": //终端上报的心跳
			handleT0Message(sess, msg)
		case "T1": //终端上报的登录请求
			return handleT1Message(sess, msg)
		case "T3": //终端实时上报
			handleT3Message(sess, msg)
		case "T4": //终端上报断电告警
			handleT4Message(sess, msg)
		case "T7": //终端上报低压告警
			handleT7Message(sess, msg)
		case "T8": //终端上报位移告警
			handleT8Message(sess, msg)
		case "T16": //里程统计信息上报
			handleT16Message(sess, msg)
		case "T17": //终端上报超速告警信息
			handleT17Message(sess, msg)
		case "T21": //终端上报震动告警
			handleT21Message(sess, msg)
		case "T41": //终端上报高温告警
			handleT41Message(sess, msg)
		case "T42": //终端上报高压告警
			handleT42Message(sess, msg)
		case "T43": //终端上报未知告警
			handleT43Message(sess, msg)
		case "T2", /*参数设置应答*/
			"T10", /*请求实时位置应答*/
			"T11", /*请求重启应答*/
			"T12", /*设防应答*/
			"T13", /*撤防应答*/
			"T14": /*参数查询应答*/
			handlePassiveMessage(sess, msg)
		}
		return true
	} else {
		return false
	}
}

func handleT0Message(sess *session.Session, msg *message.TerminalMessage) {
	feedback := message.ServerMessage{
		time.Now().Format("2006-01-02 15:04:05"),
		"S0",
		[]string{},
	}
	sess.Write([]byte(feedback.String()))
	log.Println("Feedback:", feedback.String())
}

func handleT1Message(sess *session.Session, msg *message.TerminalMessage) bool {
	if len((*msg).Params) < 6 {
		log.Println("接收到T1消息，但是参数不足")
		return false
	}

	o := mysql.GetOrm()
	var ret []orm.Params
	num, err := o.Raw("SELECT terminal.id as id,terminal.user_id as user_id,terminal.group_id as group_id,terminal_profile.imsi as imsi,terminal_profile.is_activated as is_activated,terminal_profile.expire_on as expire_on FROM terminal,terminal_profile WHERE terminal.terminal_sn = ? AND terminal.terminal_profile_id = terminal_profile.id", msg.TerminalSn).Values(&ret)
	if err == nil && num > 0 {
		if ret[0]["imsi"].(string) != (*msg).Params[3] {
			// 2表示拒绝登录—终端编号、IMEI已在平台有记录但SIM卡IMSI匹配不成功
			feedback := message.ServerMessage{
				time.Now().Format("2006-01-02 15:04:05"),
				"S1",
				[]string{"2"},
			}
			sess.Write([]byte(feedback.String()))
			log.Println("Feedback:", feedback.String())
			return false
		}

		is_activated, err := strconv.ParseInt(ret[0]["is_activated"].(string), 10, 16)
		if err != nil || is_activated != 1 {
			// 8表示未激活
			feedback := message.ServerMessage{
				time.Now().Format("2006-01-02 15:04:05"),
				"S1",
				[]string{"8"},
			}
			sess.Write([]byte(feedback.String()))
			log.Println("Feedback:", feedback.String())
			return false
		}
		expire_on, err := time.Parse("2006-01-02 15:04:05", ret[0]["expire_on"].(string))
		if err != nil || time.Now().After(expire_on) {
			// 0表示拒绝登陆--服务到期
			feedback := message.ServerMessage{
				time.Now().Format("2006-01-02 15:04:05"),
				"S1",
				[]string{"0"},
			}
			sess.Write([]byte(feedback.String()))
			log.Println("Feedback:", feedback.String())
			return false
		}
		// 1表示登陆成功
		feedback := message.ServerMessage{
			time.Now().Format("2006-01-02 15:04:05"),
			"S1",
			[]string{"1"},
		}

		sess.Write([]byte(feedback.String()))
		log.Println("Feedback:", feedback.String())

		id, _ := strconv.ParseInt(ret[0]["id"].(string), 10, 64)
		uid, _ := strconv.ParseInt(ret[0]["user_id"].(string), 10, 64)
		gid, _ := strconv.ParseInt(ret[0]["group_id"].(string), 10, 64)
		(*sess).TerminalId = id
		(*sess).UserId = uid
		(*sess).GroupId = gid
	} else {
		// 3表示拒绝登录—终端和SIM卡信息未录入
		// feedback := message.ServerMessage{
		// 	time.Now().Format("2006-01-02 15:04:05"),
		// 	"S1",
		// 	[]string{"3"},
		// }
		// sess.Write([]byte(feedback.String()))
		// log.Println("Feedback:", feedback.String())
		// return
		//
		// 为了方便测试 设备自动注册
		if err := o.Begin(); err == nil {
			var profileId int64
			if res, err := o.Raw("INSERT terminal_profile(terminal_sn,tmsisdn,pmsisdn,imsi,imei,product_code,is_activated,mileage,activate_on,expire_on,create_on,modify_on) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)",
				(*msg).TerminalSn,
				(*msg).Params[0],
				(*msg).Params[1],
				(*msg).Params[3],
				(*msg).Params[4],
				(*msg).Params[5],
				1,
				0,
				time.Now().Format("2006-01-02 15:04:05"),
				time.Now().AddDate(1, 0, 0).Format("2006-01-02 15:04:05"),
				time.Now().Format("2006-01-02 15:04:05"),
				time.Now().Format("2006-01-02 15:04:05"),
			).Exec(); err == nil {
				profileId, _ = res.LastInsertId()
			} else {
				o.Rollback()
				log.Println(err.Error())
				return false
			}

			var carrierId int64
			if res, err := o.Raw("INSERT terminal_carrier(create_on,modify_on) VALUES(?,?)",
				time.Now().Format("2006-01-02 15:04:05"),
				time.Now().Format("2006-01-02 15:04:05"),
			).Exec(); err == nil {
				carrierId, _ = res.LastInsertId()
			} else {
				o.Rollback()
				log.Println(err.Error())
				return false
			}

			if res, err := o.Raw("INSERT terminal(terminal_sn,password,user_id,group_id,terminal_profile_id,terminal_carrier_id,create_on,modify_on) VALUES(?,?,?,?,?,?,?,?)",
				(*msg).TerminalSn,
				(*msg).Params[2],
				0,
				0,
				profileId,
				carrierId,
				time.Now().Format("2006-01-02 15:04:05"),
				time.Now().Format("2006-01-02 15:04:05"),
			).Exec(); err != nil {
				o.Rollback()
				log.Println(err.Error())
				return false
			} else {
				if err := o.Commit(); err == nil {
					(*sess).TerminalId, _ = res.LastInsertId()
					(*sess).UserId = 0
					(*sess).GroupId = 0
					// 1表示登陆成功
					feedback := message.ServerMessage{
						time.Now().Format("2006-01-02 15:04:05"),
						"S1",
						[]string{"1"},
					}
					sess.Write([]byte(feedback.String()))
					log.Println("Feedback:", feedback.String())
				} else {
					log.Println(err.Error())
					return false
				}
			}
		}
	}

	// 登录只针对新设备
	if sess.State == session.SESSION_STATE_CREATED {
		// 登录成功
		if err := sessionManager.Put(sess); err != nil {
			log.Println(err.Error())
			return false
		}
	}

	return true
}

//位置上报
func handleT3Message(sess *session.Session, msg *message.TerminalMessage) {
	if len((*msg).Params) < 12 {
		log.Println("接收到T3消息，但是参数不足")
		return
	}
	//定位成功
	if (*msg).Params[0] == "1" {
		//获取数据库连接
		dbSess, err := models.GetDBSession()
		if err != nil {
			log.Println(err.Error())
			return
		}
		defer dbSess.Close()
		c := dbSess.DB(fmt.Sprintf("endville-gps-%s", time.Now().Format("200601"))).C("geo")
		//时间
		indexTimeStamp := gopkg.Index{
			Key:        []string{"-ts"},
			Unique:     false,
			DropDups:   false,
			Background: false, // See notes.
		}
		if err := c.EnsureIndex(indexTimeStamp); err != nil {
			log.Println(err.Error())
		}
		//车队
		indexGroup := gopkg.Index{
			Key:        []string{"gid"},
			Unique:     false,
			DropDups:   false,
			Background: false, // See notes.
		}
		if err := c.EnsureIndex(indexGroup); err != nil {
			log.Println(err.Error())
		}
		//地理位置
		indexGeo := gopkg.Index{
			Key:        []string{"$2d:loc"},
			Unique:     false,
			DropDups:   false,
			Background: false, // See notes.
		}
		if err := c.EnsureIndex(indexGeo); err != nil {
			log.Println(err.Error())
		}
		//准备新的GEO数据
		longitude, _ := strconv.ParseFloat((*msg).Params[2], 32)
		latitude, _ := strconv.ParseFloat((*msg).Params[4], 32)
		speed, _ := strconv.ParseFloat((*msg).Params[5], 32)
		direction, _ := strconv.ParseFloat((*msg).Params[6], 32)
		status, _ := strconv.ParseInt((*msg).Params[7], 10, 32)
		voltage, _ := strconv.ParseFloat((*msg).Params[10], 32)
		temperature, _ := strconv.ParseInt((*msg).Params[11], 10, 32)
		str := bson.NewObjectId().Hex()
		geo := models.Geo{
			Id:         str,
			GroupId:    sess.GroupId,
			UserId:     sess.UserId,
			TerminalId: sess.TerminalId,
			Location: models.Location{
				Longitude: float32(longitude),
				Latitude:  float32(latitude),
			},
			TimeStamp:   time.Now().Unix(),
			Speed:       float32(speed),
			Direction:   float32(direction),
			Status:      int(status),
			CellId:      (*msg).Params[8],
			Info:        (*msg).Params[9], // 三个数字：卫星数量、信号强度、电量
			Voltage:     float32(voltage), // 电压
			Temperature: int(temperature), // 温度
		}
		if err := c.Insert(&geo); err != nil {
			log.Println(err.Error())
		}
	}
	feedback := message.ServerMessage{
		time.Now().Format("2006-01-02 15:04:05"),
		"S3",
		[]string{},
	}
	sess.Write([]byte(feedback.String()))
	log.Println("Feedback:", feedback.String())
}

func handleT4Message(sess *session.Session, msg *message.TerminalMessage) {
	if len((*msg).Params) < 11 {
		log.Println("接收到T4消息，但是参数不足")
		return
	}

	//准备报警数据
	longitude, _ := strconv.ParseFloat((*msg).Params[2], 32)
	latitude, _ := strconv.ParseFloat((*msg).Params[4], 32)
	speed, _ := strconv.ParseFloat((*msg).Params[5], 32)
	direction, _ := strconv.ParseFloat((*msg).Params[6], 32)
	status, _ := strconv.ParseInt((*msg).Params[7], 10, 32)
	voltage, _ := strconv.ParseFloat((*msg).Params[9], 32)
	temperature, _ := strconv.ParseInt((*msg).Params[10], 10, 32)

	warning := models.Warning{
		TerminalSn:  (*sess).TerminalSn,
		TerminalId:  (*sess).TerminalId,
		UserId:      (*sess).UserId,
		GroupId:     (*sess).GroupId,
		Longitude:   float32(longitude),
		Latitude:    float32(latitude),
		Speed:       float32(speed),
		Direction:   float32(direction),
		Status:      int(status),
		CellId:      (*msg).Params[8],
		Voltage:     float32(voltage),
		Temperature: int(temperature),
		Type:        models.WARNING_TYPE_NO_POWER,
		State:       models.WARNING_STATE_NOT_SOLVED,
		Flag:        models.WARNING_FLAG_NORMAL,
	}

	mysql.AddWarning(&warning)

	feedback := message.ServerMessage{
		time.Now().Format("2006-01-02 15:04:05"),
		"S4",
		[]string{},
	}
	sess.Write([]byte(feedback.String()))
	log.Println("Feedback:", feedback.String())
}

func handleT7Message(sess *session.Session, msg *message.TerminalMessage) {
	if len((*msg).Params) < 11 {
		log.Println("接收到T7消息，但是参数不足")
		return
	}

	//准备报警数据
	longitude, _ := strconv.ParseFloat((*msg).Params[2], 32)
	latitude, _ := strconv.ParseFloat((*msg).Params[4], 32)
	speed, _ := strconv.ParseFloat((*msg).Params[5], 32)
	direction, _ := strconv.ParseFloat((*msg).Params[6], 32)
	status, _ := strconv.ParseInt((*msg).Params[7], 10, 32)
	voltage, _ := strconv.ParseFloat((*msg).Params[9], 32)
	temperature, _ := strconv.ParseInt((*msg).Params[10], 10, 32)

	warning := models.Warning{
		TerminalSn:  (*sess).TerminalSn,
		TerminalId:  (*sess).TerminalId,
		UserId:      (*sess).UserId,
		GroupId:     (*sess).GroupId,
		Longitude:   float32(longitude),
		Latitude:    float32(latitude),
		Speed:       float32(speed),
		Direction:   float32(direction),
		Status:      int(status),
		CellId:      (*msg).Params[8],
		Voltage:     float32(voltage),
		Temperature: int(temperature),
		Type:        models.WARNING_TYPE_LOW_VOLTAGE,
		State:       models.WARNING_STATE_NOT_SOLVED,
		Flag:        models.WARNING_FLAG_NORMAL,
	}

	mysql.AddWarning(&warning)
	feedback := message.ServerMessage{
		time.Now().Format("2006-01-02 15:04:05"),
		"S7",
		[]string{},
	}
	sess.Write([]byte(feedback.String()))
	log.Println("Feedback:", feedback.String())
}

func handleT8Message(sess *session.Session, msg *message.TerminalMessage) {
	if len((*msg).Params) < 11 {
		log.Println("接收到T8消息，但是参数不足")
		return
	}

	//准备报警数据
	longitude, _ := strconv.ParseFloat((*msg).Params[2], 32)
	latitude, _ := strconv.ParseFloat((*msg).Params[4], 32)
	speed, _ := strconv.ParseFloat((*msg).Params[5], 32)
	direction, _ := strconv.ParseFloat((*msg).Params[6], 32)
	status, _ := strconv.ParseInt((*msg).Params[7], 10, 32)
	voltage, _ := strconv.ParseFloat((*msg).Params[9], 32)
	temperature, _ := strconv.ParseInt((*msg).Params[10], 10, 32)

	warning := models.Warning{
		TerminalSn:  (*sess).TerminalSn,
		TerminalId:  (*sess).TerminalId,
		UserId:      (*sess).UserId,
		GroupId:     (*sess).GroupId,
		Longitude:   float32(longitude),
		Latitude:    float32(latitude),
		Speed:       float32(speed),
		Direction:   float32(direction),
		Status:      int(status),
		CellId:      (*msg).Params[8],
		Voltage:     float32(voltage),
		Temperature: int(temperature),
		Type:        models.WARNING_TYPE_MOVE,
		State:       models.WARNING_STATE_NOT_SOLVED,
		Flag:        models.WARNING_FLAG_NORMAL,
	}

	mysql.AddWarning(&warning)

	feedback := message.ServerMessage{
		time.Now().Format("2006-01-02 15:04:05"),
		"S8",
		[]string{},
	}
	sess.Write([]byte(feedback.String()))
	log.Println("Feedback:", feedback.String())
}

func handleT16Message(sess *session.Session, msg *message.TerminalMessage) {
	if len((*msg).Params) < 1 {
		log.Println("接收到T16消息，但是参数不足")
		return
	}
	if miles, parseErr := strconv.ParseInt((*msg).Params[0], 10, 64); parseErr != nil {
		log.Println(parseErr.Error())
	} else {
		if miles > 0 {
			o := mysql.GetOrm()
			if _, err := o.Raw("UPDATE terminal_profile SET mileage = mileage + ? WHERE terminal_sn = ?",
				miles,
				(*msg).TerminalSn,
			).Exec(); err != nil {
				log.Println(err.Error())
			}
			if _, err := o.Raw("INSERT mileage(terminal_id,user_id,group_id,mileage,record_on) VALUES (?,?,?,?,?)",
				(*sess).TerminalId,
				(*sess).UserId,
				(*sess).GroupId,
				miles,
				time.Now().Format("2006-01-02 15:04:05"),
			).Exec(); err != nil {
				log.Println(err.Error())
			}
		}
	}
	feedback := message.ServerMessage{
		time.Now().Format("2006-01-02 15:04:05"),
		"S16",
		[]string{},
	}
	sess.Write([]byte(feedback.String()))
	log.Println("Feedback:", feedback.String())
}

func handleT17Message(sess *session.Session, msg *message.TerminalMessage) {
	if len((*msg).Params) < 11 {
		log.Println("接收到T17消息，但是参数不足")
		return
	}
	//准备报警数据
	longitude, _ := strconv.ParseFloat((*msg).Params[2], 32)
	latitude, _ := strconv.ParseFloat((*msg).Params[4], 32)
	speed, _ := strconv.ParseFloat((*msg).Params[5], 32)
	direction, _ := strconv.ParseFloat((*msg).Params[6], 32)
	status, _ := strconv.ParseInt((*msg).Params[7], 10, 32)
	voltage, _ := strconv.ParseFloat((*msg).Params[9], 32)
	temperature, _ := strconv.ParseInt((*msg).Params[10], 10, 32)

	warning := models.Warning{
		TerminalSn:  (*sess).TerminalSn,
		TerminalId:  (*sess).TerminalId,
		UserId:      (*sess).UserId,
		GroupId:     (*sess).GroupId,
		Longitude:   float32(longitude),
		Latitude:    float32(latitude),
		Speed:       float32(speed),
		Direction:   float32(direction),
		Status:      int(status),
		CellId:      (*msg).Params[8],
		Voltage:     float32(voltage),
		Temperature: int(temperature),
		Type:        models.WARNING_TYPE_OVER_SPEED,
		State:       models.WARNING_STATE_NOT_SOLVED,
		Flag:        models.WARNING_FLAG_NORMAL,
	}

	mysql.AddWarning(&warning)

	feedback := message.ServerMessage{
		time.Now().Format("2006-01-02 15:04:05"),
		"S17",
		[]string{},
	}
	sess.Write([]byte(feedback.String()))
	log.Println("Feedback:", feedback.String())
}

func handleT21Message(sess *session.Session, msg *message.TerminalMessage) {
	if len((*msg).Params) < 1 {
		log.Println("接收到T21消息，但是参数不足")
		return
	}

	//准备报警数据
	longitude, _ := strconv.ParseFloat((*msg).Params[2], 32)
	latitude, _ := strconv.ParseFloat((*msg).Params[4], 32)
	speed, _ := strconv.ParseFloat((*msg).Params[5], 32)
	direction, _ := strconv.ParseFloat((*msg).Params[6], 32)
	status, _ := strconv.ParseInt((*msg).Params[7], 10, 32)
	voltage, _ := strconv.ParseFloat((*msg).Params[9], 32)
	temperature, _ := strconv.ParseInt((*msg).Params[10], 10, 32)

	warning := models.Warning{
		TerminalSn:  (*sess).TerminalSn,
		TerminalId:  (*sess).TerminalId,
		UserId:      (*sess).UserId,
		GroupId:     (*sess).GroupId,
		Longitude:   float32(longitude),
		Latitude:    float32(latitude),
		Speed:       float32(speed),
		Direction:   float32(direction),
		Status:      int(status),
		CellId:      (*msg).Params[8],
		Voltage:     float32(voltage),
		Temperature: int(temperature),
		Type:        models.WARNING_TYPE_VIBRATE,
		State:       models.WARNING_STATE_NOT_SOLVED,
		Flag:        models.WARNING_FLAG_NORMAL,
	}

	mysql.AddWarning(&warning)

	feedback := message.ServerMessage{
		time.Now().Format("2006-01-02 15:04:05"),
		"S21",
		[]string{},
	}
	sess.Write([]byte(feedback.String()))
	log.Println("Feedback:", feedback.String())
}

func handleT41Message(sess *session.Session, msg *message.TerminalMessage) {
	if len((*msg).Params) < 1 {
		log.Println("接收到T41消息，但是参数不足")
		return
	}

	//准备报警数据
	longitude, _ := strconv.ParseFloat((*msg).Params[2], 32)
	latitude, _ := strconv.ParseFloat((*msg).Params[4], 32)
	speed, _ := strconv.ParseFloat((*msg).Params[5], 32)
	direction, _ := strconv.ParseFloat((*msg).Params[6], 32)
	status, _ := strconv.ParseInt((*msg).Params[7], 10, 32)
	voltage, _ := strconv.ParseFloat((*msg).Params[9], 32)
	temperature, _ := strconv.ParseInt((*msg).Params[10], 10, 32)

	warning := models.Warning{
		TerminalSn:  (*sess).TerminalSn,
		TerminalId:  (*sess).TerminalId,
		UserId:      (*sess).UserId,
		GroupId:     (*sess).GroupId,
		Longitude:   float32(longitude),
		Latitude:    float32(latitude),
		Speed:       float32(speed),
		Direction:   float32(direction),
		Status:      int(status),
		CellId:      (*msg).Params[8],
		Voltage:     float32(voltage),
		Temperature: int(temperature),
		Type:        models.WARNING_TYPE_HIGH_TEMPERATURE,
		State:       models.WARNING_STATE_NOT_SOLVED,
		Flag:        models.WARNING_FLAG_NORMAL,
	}

	mysql.AddWarning(&warning)

	feedback := message.ServerMessage{
		time.Now().Format("2006-01-02 15:04:05"),
		"S41",
		[]string{},
	}
	sess.Write([]byte(feedback.String()))
	log.Println("Feedback:", feedback.String())
}

func handleT42Message(sess *session.Session, msg *message.TerminalMessage) {
	if len((*msg).Params) < 11 {
		log.Println("接收到T42消息，但是参数不足")
		return
	}

	//准备报警数据
	longitude, _ := strconv.ParseFloat((*msg).Params[2], 32)
	latitude, _ := strconv.ParseFloat((*msg).Params[4], 32)
	speed, _ := strconv.ParseFloat((*msg).Params[5], 32)
	direction, _ := strconv.ParseFloat((*msg).Params[6], 32)
	status, _ := strconv.ParseInt((*msg).Params[7], 10, 32)
	voltage, _ := strconv.ParseFloat((*msg).Params[9], 32)
	temperature, _ := strconv.ParseInt((*msg).Params[10], 10, 32)

	warning := models.Warning{
		TerminalSn:  (*sess).TerminalSn,
		TerminalId:  (*sess).TerminalId,
		UserId:      (*sess).UserId,
		GroupId:     (*sess).GroupId,
		Longitude:   float32(longitude),
		Latitude:    float32(latitude),
		Speed:       float32(speed),
		Direction:   float32(direction),
		Status:      int(status),
		CellId:      (*msg).Params[8],
		Voltage:     float32(voltage),
		Temperature: int(temperature),
		Type:        models.WARNING_TYPE_LOW_TEMPERATURE,
		State:       models.WARNING_STATE_NOT_SOLVED,
		Flag:        models.WARNING_FLAG_NORMAL,
	}

	mysql.AddWarning(&warning)

	feedback := message.ServerMessage{
		time.Now().Format("2006-01-02 15:04:05"),
		"S42",
		[]string{},
	}
	sess.Write([]byte(feedback.String()))
	log.Println("Feedback:", feedback.String())
}

func handleT43Message(sess *session.Session, msg *message.TerminalMessage) {
	if len((*msg).Params) < 11 {
		log.Println("接收到T43消息，但是参数不足")
		return
	}

	//准备报警数据
	longitude, _ := strconv.ParseFloat((*msg).Params[2], 32)
	latitude, _ := strconv.ParseFloat((*msg).Params[4], 32)
	speed, _ := strconv.ParseFloat((*msg).Params[5], 32)
	direction, _ := strconv.ParseFloat((*msg).Params[6], 32)
	status, _ := strconv.ParseInt((*msg).Params[7], 10, 32)
	voltage, _ := strconv.ParseFloat((*msg).Params[9], 32)
	temperature, _ := strconv.ParseInt((*msg).Params[10], 10, 32)

	warning := models.Warning{
		TerminalSn:  (*sess).TerminalSn,
		TerminalId:  (*sess).TerminalId,
		UserId:      (*sess).UserId,
		GroupId:     (*sess).GroupId,
		Longitude:   float32(longitude),
		Latitude:    float32(latitude),
		Speed:       float32(speed),
		Direction:   float32(direction),
		Status:      int(status),
		CellId:      (*msg).Params[8],
		Voltage:     float32(voltage),
		Temperature: int(temperature),
		Type:        models.WARNING_TYPE_UNKONW,
		State:       models.WARNING_STATE_NOT_SOLVED,
		Flag:        models.WARNING_FLAG_NORMAL,
	}

	mysql.AddWarning(&warning)

	feedback := message.ServerMessage{
		time.Now().Format("2006-01-02 15:04:05"),
		"S43",
		[]string{},
	}
	sess.Write([]byte(feedback.String()))
	log.Println("Feedback:", feedback.String())
}

// 被动
func handlePassiveMessage(sess *session.Session, msg *message.TerminalMessage) {
	if (*sess).FeedbackChannel != nil {
		log.Println("Push feedback to channel.")
		select {
		case (*sess).FeedbackChannel <- msg:
			log.Println("Pushed.")
			break
		case <-time.After(3 * time.Second):
			// 将msg暂时保存？
			log.Println("Push failed, timeout.")
			break
		}
	} else {
		log.Println("Feedback channel is nil.")
	}
}

func init() {
	sessionManager = session.New()
}
