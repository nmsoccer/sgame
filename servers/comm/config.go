package comm

import(
    "sgame/lib/log"
    "sgame/lib/proc"
    "time"
    "os"
    "os/signal"
    "syscall"	
    "fmt"
    "encoding/json"
)

const (
	TIME_EPOLL_BASE int64 = 1577808000 //2020-01-01 00:00:00

	TIME_FORMAT_SEC="2006-01-02 15:04:05"
	TIME_FORMAT_MILL="2006-01-02 15:04:05.000"
	TIME_FORMAT_MICR="2006-01-02 15:04:05.000000"
	TIME_FORMAT_NANO="2006-01-02 15:04:05.000000000"

	DEFAULT_SERVER_SLEEP_IDLE=5 //ms. server sleeps when idle 
	
    INFO_EXIT = iota //0 server exit
    INFO_USR1 //1 server reload config
    INFO_USR2 //2 reload tables
)


type CommConfig struct {
    StartTs int64
    Log log.LogHeader
    LockFile *os.File
	Proc proc.ProcHeader
	ChSig chan os.Signal
	ChInfo chan int
	PeerStats map[int] int64 //peer [procid]->heart_beat_ts
	TickPool *TickPool
    ServerCfg interface{} //server *config if assigend
}


func InitCommConfig(log_file string , name_space string , proc_id int) *CommConfig {
	pconfig := new(CommConfig);
	if pconfig == nil {
		fmt.Printf("InitCommConfig failed!\n");
		return nil;
	}
	
	//start
	pconfig.StartTs = time.Now().Unix();
	
	//log
    lp := log.OpenLog(log_file);
    if lp == nil {
    	fmt.Printf("open log %s failed!\n" , log_file);
    	return nil;
    }
    pconfig.Log = lp;
    
    //open bridge
    if proc_id>0 {
        p := proc.Open(name_space , proc_id);
        if p == nil {
    	    lp.Err("open bridge <%s:%d> failed!" , name_space , proc_id);
    	    return nil;
        }
        pconfig.Proc = p;
        lp.Info("open proc bridge <%s:%d> success!" , name_space , proc_id);
    }
	
		
	//signal
	pconfig.ChSig = make(chan os.Signal , 16);
	signal.Notify(pconfig.ChSig , syscall.SIGINT , syscall.SIGTERM , syscall.SIGUSR1 , syscall.SIGUSR2);
	
	//msg
	pconfig.ChInfo = make(chan int , 16);	
		
	//peer stats
	pconfig.PeerStats = make(map[int]int64);

	//tick pool
	pconfig.TickPool = NewTickPool(pconfig);
	if pconfig.TickPool == nil {
		lp.Err("new tick pool failed!");
		return nil;
	}

	return pconfig;
}

func LoadJsonFile(config_file string , file_config interface{} , pconfig *CommConfig) bool{
	var _func_ = "<LoadJsonFile>";
	var log log.LogHeader = nil;
	if pconfig != nil {
		log = pconfig.Log;
	}
	
	file , err := os.Open(config_file);
	if err != nil {
		fmt.Printf("%s open %s failed! err:%v\n", _func_ , config_file , err);
		if log != nil {
			log.Err("%s open %s failed! err:%v", _func_ , config_file , err);
		}
		return false;
	}
	defer file.Close();
	
	//decoder
	var decoder *json.Decoder;
	decoder = json.NewDecoder(file);
	if decoder == nil {
		fmt.Printf("%s new json decoder %s failed!\n" , _func_ , config_file);
		if log != nil {
			log.Err("%s new json decoder %s failed!" , _func_ , config_file);
		}
		return false;
	}
	
	//decode
	err = decoder.Decode(file_config);
	if err != nil {
		fmt.Printf("%s decode config failed! err:%v\n", _func_ , err);
		if log != nil {
			log.Err("%s decode config failed! err:%v", _func_ , err);
		}
		return false;
	}
	fmt.Printf("FileConfig:%v\n", file_config);
	if log != nil {
		log.Info("%s load %s success!config:%v", _func_ , config_file , file_config);
	}
	return true;
}

//generate local id
var seq uint16 = 1;
func GenerateLocalId(wid int16) int64 {
	var id int64 = 0
	curr_ts := time.Now().Unix() //
    diff := curr_ts - TIME_EPOLL_BASE;

    seq += 1;
    if seq >= 65530 {
    	seq = 1;
	}

    id = ((int64(seq) & 0xFFFF) << 47) | ((int64(wid) & 0xFFFF) << 31) | (diff & 0x7FFFFFFF);
	return id;
}