package main

import (
	"github.com/Unknwon/goconfig"
	"gpsgate/service"
	"log"
	"runtime"
	"sync"
)

// 主进程阻塞
var waitGroup sync.WaitGroup

var (
	// 监听GPS端口
	LISTEN_PORT = 8500
	// 监听管理端口
	MANAGE_PORT = 8501
	// RPC管理端口
	RPC_PORT = 8502
)

func main() {
	// 充分利用多核CPU
	runtime.GOMAXPROCS(runtime.NumCPU())
	waitGroup.Add(1)

	// 监听终端端口
	go service.StartTerminalListeningService(LISTEN_PORT)
	log.Println("Terminal Listening ", LISTEN_PORT, "...")

	// 监听管理端口
	// go service.StartTerminalManageService(MANAGE_PORT)
	// log.Println("Manage Listening ", MANAGE_PORT, "...")

	// 监听管理端口
	go service.StartRPCTerminalManageService(RPC_PORT)
	log.Println("RPC Listening ", RPC_PORT, "...")

	// 主进程阻塞
	waitGroup.Wait()
}

func init() {
	if cfg, err := goconfig.LoadConfigFile("config.ini"); err == nil {
		if value, err := cfg.Int(goconfig.DEFAULT_SECTION, "listen_port"); err == nil {
			LISTEN_PORT = value
		}
		if value, err := cfg.Int(goconfig.DEFAULT_SECTION, "manage_port"); err == nil {
			MANAGE_PORT = value
		}
		if value, err := cfg.Int(goconfig.DEFAULT_SECTION, "rpc_port"); err == nil {
			RPC_PORT = value
		}
	} else {
		log.Println("读取配置文件失败[conf.ini]")
		return
	}
}
