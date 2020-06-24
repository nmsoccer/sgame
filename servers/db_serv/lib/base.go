package lib

import(
    "sgame/servers/comm"
    "time"
    "os"
    "fmt"
)


type FileConfig struct {
//	ProcName string `json:"proc_name"`
	LogicServs []int `json:"logic_servs"` //logic serv set
	LogFile string `json:"log_file"`
	RedisOpen int `json:"redis_open"`
	RedisAddr string `json:"redis_addr"`
	MaxConn int `json:"max_conn"` //max redis-conn of process
	NormalConn int `json:"normal_conn"` //normal redis-conn
	AuthPass string `json:"auth_pass"`
	ManageAddr []string `json:"manage_addr"`
}


type Config struct {
	NameSpace string
	ProcId int
	ProcName string
    ConfigFile string
	FileConfig *FileConfig    
	Comm *comm.CommConfig;
	RedisClient *comm.RedisClient;	
	ReportServ *comm.ReportServ; //report to manger
}

//Comm Config Setting
func CommSet(pconfig *Config) bool {
	var _func_ = "<CommSet>";
	//daemonize
	comm.Daemonize();
	
	//file config
	pconfig.FileConfig = new(FileConfig);
	if pconfig.FileConfig == nil {
		fmt.Printf("new FileConfig failed!\n");
		return false;
	}
	
	//load file config
	if comm.LoadJsonFile(pconfig.ConfigFile , pconfig.FileConfig , nil) != true {
		fmt.Printf("%s failed!\n", _func_);		
		return false;
	}
	
	//comm config
	pconfig.Comm = comm.InitCommConfig(pconfig.FileConfig.LogFile , pconfig.NameSpace , pconfig.ProcId);
	
	//lock uniq
	if comm.LockUniqFile(pconfig.Comm , pconfig.NameSpace , pconfig.ProcId , pconfig.ProcName) == false {
		pconfig.Comm.Log.Err("%s lock uniq file failed!" , _func_);
		return false;
	}	
	return true;
}

//Self Proc Setting
func SelfSet(pconfig *Config) bool {
	var _func_ = "<SelfSet>";
	log := pconfig.Comm.Log;
	
	//connect to redis
	if pconfig.FileConfig.RedisOpen == 1 {
	    pclient := OpenRedis(pconfig);
	    if pclient==nil {
		    log.Err("%s failed! open redis addr:%s fail!" , _func_ , pconfig.FileConfig.RedisAddr);
		    return false;
	    }
	    pconfig.RedisClient = pclient;
	}
	
	//start report serv
	pconfig.ReportServ = comm.StartReport(pconfig.Comm , pconfig.ProcId , pconfig.ProcName , pconfig.FileConfig.ManageAddr , comm.REPORT_METHOD_ALL);
    pconfig.ReportServ.Report(comm.REPORT_PROTO_SERVER_START , time.Now().Unix() , "" , nil);
	 
	return true;
}


//Server Exit
func ServerExit(pconfig *Config) {
	//close proc	
	if pconfig.Comm.Proc != nil {
		pconfig.Comm.Proc.Close();
	}
	
	//close redis
	if pconfig.RedisClient != nil {
	    pconfig.RedisClient.Close(pconfig.Comm);
	    time.Sleep(time.Second);
	}
	
	//close report_serv
	if pconfig.ReportServ != nil {
	    pconfig.ReportServ.Close();
	}
	
	//unlock uniq
	comm.UnlockUniqFile(pconfig.Comm , pconfig.NameSpace , pconfig.ProcId , pconfig.ProcName);
	
	//close log
	if pconfig.Comm.Log != nil {
		pconfig.Comm.Log.Info("%s" , "server exit...");
		pconfig.Comm.Log.Close();
	}
	time.Sleep(1 * time.Second);
	os.Exit(0);	
}

//Main Proc
func ServerStart(pconfig *Config) {
	var log = pconfig.Comm.Log;
	var default_sleep = time.Duration(comm.DEFAULT_SERVER_SLEEP_IDLE);
	log.Info("%s starts---%v" , pconfig.ProcName , os.Args);
	
	//each support routine
	go comm.HandleSignal(pconfig.Comm);
	
	//main loop
	for {
		//handle info
		handle_info(pconfig);
		
		//recv pkg
		RecvMsg(pconfig);
		
		//tick
		handle_tick(pconfig);
		
		//sleep		
		time.Sleep(time.Millisecond * default_sleep);     
	}
}

/*----------------Static Func--------------------*/
func handle_info(pconfig *Config) {
	log := pconfig.Comm.Log;	
	select {
		case m := <- pconfig.Comm.ChInfo:
		    switch m {
		    	case comm.INFO_EXIT:
		    	    ServerExit(pconfig);
		    	case comm.INFO_USR1:
		    	    log.Info(">>reload config!");
		    	    comm.LoadJsonFile(pconfig.ConfigFile , pconfig.FileConfig , pconfig.Comm);
		    	case comm.INFO_USR2:
		    	    log.Info(">>info usr2");
		    	default:
		    	    pconfig.Comm.Log.Info("unknown msg");        
		    }
		default:
		    break;
	}	
}

//each ticker
func handle_tick(pconfig *Config) {
    SendHeartBeatMsg(pconfig);
    HeartBeatToRedis(pconfig);
    ReportSyncServer(pconfig);    	
}