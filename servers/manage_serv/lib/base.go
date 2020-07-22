package lib

import (
	"fmt"
	"os"
	"sgame/servers/comm"
	"time"
)

type WatchClient struct {
	ProcId   int
	ProcName string
	Stat     PeerStat
}

type FileConfig struct {
	ListenAddr    string        `json:"listen_addr"` //listen report
	HttpAddr      string        `json:"http_addr"`   //listen http request
	LogFile       string        `json:"log_file"`
	ClientList    []interface{} `json:"client_list"`
	HeartTimeout  int           `json:"heart_timeout"`
	ReloadTimeout int           `json:"reload_timeout"`
}

type Config struct {
	NameSpace  string
	ProcId     int
	ProcName   string
	ConfigFile string
	FileConfig *FileConfig
	Comm       *comm.CommConfig
	WatchMap   map[int]*WatchClient
	Name2Id    map[string]int
	Recver     *ReportRecver
	Panel      *PanelServ
}

//Comm Config Setting
func CommSet(pconfig *Config) bool {
	var _func_ = "<CommSet>"
	//daemonize
	comm.Daemonize()

	//file config
	pconfig.FileConfig = new(FileConfig)
	if pconfig.FileConfig == nil {
		fmt.Printf("new FileConfig failed!\n")
		return false
	}

	//load file config
	if comm.LoadJsonFile(pconfig.ConfigFile, pconfig.FileConfig, nil) != true {
		fmt.Printf("%s failed!\n", _func_)
		return false
	}

	//comm config
	pconfig.Comm = comm.InitCommConfig(pconfig.FileConfig.LogFile, pconfig.NameSpace, pconfig.ProcId)

	//lock uniq
	if comm.LockUniqFile(pconfig.Comm, pconfig.NameSpace, pconfig.ProcId, pconfig.ProcName) == false {
		pconfig.Comm.Log.Err("%s lock uniq file failed!", _func_)
		return false
	}
	return true
}

//Self Proc Setting
func SelfSet(pconfig *Config) bool {
	var _func_ = "<SelfSet>"
	log := pconfig.Comm.Log

	//parse client
	if !ParseClientList(pconfig) {
		log.Err("%s parse client list failed!", _func_)
		return false
	}

	//start recver
	pconfig.Recver = StartRecver(pconfig)
	if pconfig.Recver == nil {
		log.Err("%s failed! start recver fail!", _func_)
		return false
	}

	//start panelserv
	pconfig.Panel = StartPanel(pconfig)
	if pconfig.Panel == nil {
		log.Err("%s failed! start panel fail!", _func_)
		return false
	}

	//start tcp serv to listen clients
	log.Info("%s done", _func_)

	return true
}

func ServerExit(pconfig *Config) {
	//close proc
	if pconfig.Comm.Proc != nil {
		pconfig.Comm.Proc.Close()
	}

	//close recver
	if pconfig.Recver != nil {
		pconfig.Recver.Close()
	}

	//unlock uniq
	comm.UnlockUniqFile(pconfig.Comm, pconfig.NameSpace, pconfig.ProcId, pconfig.ProcName)

	//close log
	if pconfig.Comm.Log != nil {
		pconfig.Comm.Log.Info("%s", "server exit...")
		pconfig.Comm.Log.Close()
	}
	time.Sleep(1 * time.Second)
	os.Exit(0)
}

//Main Proc
func ServerStart(pconfig *Config) {
	var log = pconfig.Comm.Log
	var default_sleep = time.Duration(comm.DEFAULT_SERVER_SLEEP_IDLE)
	log.Info("%s starts---%v", pconfig.ProcName, os.Args)

	//each support routine
	go comm.HandleSignal(pconfig.Comm)

	//main loop
	for {
		//handle info
		handle_info(pconfig)

		//recv pkg
		//RecvMsg(pconfig);

		//tick
		handle_tick(pconfig)

		//sleep
		time.Sleep(time.Millisecond * default_sleep)
	}
}

/*----------------Static Func--------------------*/
//each ticker
func handle_tick(pconfig *Config) {
	//SendHeartBeatMsg(pconfig);
}

func handle_info(pconfig *Config) {
	log := pconfig.Comm.Log
	select {
	case m := <-pconfig.Comm.ChInfo:
		switch m {
		case comm.INFO_EXIT:
			ServerExit(pconfig)
		case comm.INFO_USR1:
			log.Info(">>reload config!")
			comm.LoadJsonFile(pconfig.ConfigFile, pconfig.FileConfig, pconfig.Comm)
		case comm.INFO_USR2:
			log.Info(">>info usr2")
		default:
			pconfig.Comm.Log.Info("unknown msg")
		}
	default:
		break
	}
}
