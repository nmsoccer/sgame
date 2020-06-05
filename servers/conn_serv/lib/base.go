package lib

import(
    "sgame/servers/comm"
    "time"
    "os"	
    "fmt"
)

type FileConfig struct {
	ProcName string `json:"proc_name"`
	LogicServ int `json:"logic_serv"`
	LogFile string `json:"log_file"`
	MaxConn int `json:"max_conn"`
	ListenAddr string `json:"listen_addr"`
}


type Config struct {
	NameSpace string
	ProcId int
	ConfigFile string
	FileConfig *FileConfig
	Comm *comm.CommConfig;
	TcpServ *comm.TcpServ;
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
	if comm.LockUniqFile(pconfig.Comm , pconfig.NameSpace , pconfig.ProcId) == false {
		pconfig.Comm.Log.Err("%s lock uniq file failed!" , _func_);
		return false;
	}	
	return true;
}

//Self Proc Setting
func SelfSet(pconfig *Config) bool {
	var _func_ = "<SelfSet>";
	log := pconfig.Comm.Log;
	
	//start tcp serv to listen clients
	pserv := comm.StartTcpServ(pconfig.Comm , pconfig.FileConfig.ListenAddr  , pconfig.FileConfig.MaxConn);
	if pserv == nil {
	    log.Err("%s fail! start_tcp_serv at %s failed!" , _func_ , pconfig.FileConfig.ListenAddr);
	    return false;	
	}
	pconfig.TcpServ = pserv;
	
	return true;
}

func ServerExit(pconfig *Config) {
	//close proc	
	if pconfig.Comm.Proc != nil {
		pconfig.Comm.Proc.Close();
	}
	
	//close tcp_serv
	pconfig.TcpServ.Close(pconfig.Comm);
	
	//unlock uniq
	comm.UnlockUniqFile(pconfig.Comm , pconfig.NameSpace , pconfig.ProcId);
	
	//close log
	if pconfig.Comm.Log != nil {
		pconfig.Comm.Log.Info("%s" , "server exit...");
		pconfig.Comm.Log.Close();
	}
	os.Exit(0);	
}


//Main Proc
func ServerStart(pconfig *Config) {
	var log = pconfig.Comm.Log;
	log.Info("%s starts---%v" , pconfig.FileConfig.ProcName , os.Args);
	
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
		
		//read client
		ReadClients(pconfig);
		
		//sleep		
		time.Sleep(time.Millisecond * 10);     
	}
}

/*----------------Static Func--------------------*/
//each ticker
func handle_tick(pconfig *Config) {
    SendHeartBeatMsg(pconfig);    	
}

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