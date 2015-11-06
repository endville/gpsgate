package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var maxClient int32 = 100
var count int32 = 1
var successCount int32
var waitGroup sync.WaitGroup
var mutex sync.Mutex

// var serverAddr string = "61.153.22.147:8500"
var serverAddr string = "190.168.251.163:8500"

// var serverAddr string = "127.0.0.1:8500"
var terminalSNPrefix string
var responseCh chan string
var snFlag string

func main() {
	if len(os.Args) > 1 {
		snFlag = os.Args[1]
	}
	fmt.Println(snFlag)
	waitGroup.Add(1)

	go func() {
		for {
			select {
			case res := <-responseCh:
				log.Printf("From Server:%s\r\n", res)
			}
		}
	}()
	for {
		if count <= maxClient {
			if count%100 == 0 {
				log.Printf("尝试连接数达到%d\n", count)
			}
			go makeClient(
				responseCh,
				fmt.Sprintf("%s%03d", "测试_"+snFlag+"_"+terminalSNPrefix, count),
				fmt.Sprintf("%08d", count),
				fmt.Sprintf("1%07d", count),
				"123456",
				fmt.Sprintf("460%012d", count),
				fmt.Sprintf("355%012d", count),
				"YD",
			)
			time.Sleep(456 * time.Millisecond)
			continue
		}
		time.Sleep(5 * time.Second)
	}
	waitGroup.Wait()
}

func makeClient(ch chan string, terminalSn, tMsisdn, pMsisdn, passwd, tImsi, tImei, product string) {
	atomic.AddInt32(&count, 1)
	defer atomic.AddInt32(&count, -1)
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Printf("Dial error: %s\n", err)
		return
	}

	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	defer func() {
		atomic.AddInt32(&successCount, -1)
		conn.Close()
	}()

	atomic.AddInt32(&successCount, 1)
	log.Printf("已连接数为%d\n", successCount)

	// 先登录
	T1 := fmt.Sprintf("[%s,1,V3.0,%s,T1,%s,%s,%s,%s,%s,%s]", time.Now().Format("2006-01-02 15:04:05"), terminalSn, tMsisdn, pMsisdn, passwd, tImsi, tImei, product)
	_, writeErr := conn.Write([]byte(T1))
	if writeErr != nil {
		log.Println(err.Error())
		return
	} else {
		// 数据缓冲区大小
		databuf := make([]byte, 256)

		n, readErr := conn.Read(databuf)
		if readErr != nil {
			log.Println(err.Error())
			return
		} else {
			ch <- string(databuf[:n])
		}
	}

	alerts := []string{"4", "7", "8", "17", "21", "41", "42", "43"}

	longitude := 121.0
	latitude := 29.0
	// 发送GPS定位包
	for {
		var T string
		randAction := rand.Int31n(20)
		switch randAction {
		case 1:
			miles := rand.Int31n(2000)
			T = fmt.Sprintf("[%s,1,V3.0,%s,T16,%d]", time.Now().Format("2006-01-02 15:04:05"), terminalSn, miles)
		case 0, 6, 10:
			difLng := rand.Float64()/100 - 0.005
			longitude += difLng
			difLat := rand.Float64()/100 - 0.005
			latitude += difLat
			alert := alerts[rand.Int31n(int32(len(alerts)))]
			T = fmt.Sprintf("[%s,1,V3.0,%s,T%s,1,E,%f,N,%f,50.6,270.5,1,460:00:10101:03633,48.02,23000]", time.Now().Format("2006-01-02 15:04:05"), terminalSn, alert, longitude, latitude)
		default: //T3
			difLng := rand.Float64()/10 - 0.05
			longitude += difLng
			difLat := rand.Float64()/10 - 0.05
			latitude += difLat
			T = fmt.Sprintf("[%s,1,V3.0,%s,T3,1,E,%f,N,%f,50.6,270.5,1,460:00:10101:03633,669,48.02,23000]", time.Now().Format("2006-01-02 15:04:05"), terminalSn, longitude, latitude)
		}
		_, writeErr := conn.Write([]byte(T))
		if writeErr != nil {
			log.Println(err.Error())
			return
		} else {
			// 数据缓冲区大小
			databuf := make([]byte, 256)
			n, readErr := conn.Read(databuf)
			if readErr != nil {
				log.Println(err.Error())
				return
			} else {
				ch <- string(databuf[:n])
			}
		}

		randSleep := rand.Int31n(15) + 3
		log.Println(terminalSn, "sleep", randSleep, "s.")
		time.Sleep(time.Duration(randSleep) * time.Second)
	}
}

func init() {
	responseCh = make(chan string, 100)
	interfaces, err := net.Interfaces()
	if err != nil {
		panic("Poor soul, here is what you got: " + err.Error())
	}
	for _, inter := range interfaces {
		terminalSNPrefix = inter.HardwareAddr.String() //获取本机MAC地址
		terminalSNPrefix = strings.Replace(terminalSNPrefix, ":", "", -1)
		break
	}

	rand.Seed(time.Now().Unix())
}
