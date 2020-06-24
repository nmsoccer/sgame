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
	DEFAULT_SERVER_SLEEP_IDLE=5 //ms. server sleeps when idle 
	
    INFO_EXIT = iota //0 server exit
    INFO_USR1 //1 server reload config
    INFO_USR2 //2 reload tables
)


type CommConfig struct {
    StartTs int64
    Log log.LogHeader
	Proc proc.ProcHeader
	ChSig chan os.Signal
	ChInfo chan int
	PeerStats map[int] int64 //peer [procid]->heart_beat_ts	
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