package lib

import (
    "net"
    "time"
    "sgame/servers/comm"
)

const (
	MAX_RECV_PER_TICK = 10
)



type PeerStat struct{
	addr *net.UDPAddr
	StartTime time.Time
	HeartBeat time.Time
	ConnStr string
	ConnNum int
}


type ReportRecver struct {	
	config *Config;
	local_addr string
	recv_conn *net.UDPConn
	exit_ch chan bool	
}


//parse FileConfig.ClientList --> Config.WatchMap
func ParseClientList(pconfig *Config) bool {
	var _func_ = "ParseClientList";
	log := pconfig.Comm.Log;
	
	//init
	log.Info("%s client list:%v" , _func_ , pconfig.FileConfig.ClientList);
	pconfig.WatchMap = make(map[int]*WatchClient);
	
	//psarse
	for i:=0; i<len(pconfig.FileConfig.ClientList); i++ {
		//new
		pwatch := new(WatchClient);
		if pwatch == nil {
			log.Err("%s new client failed!" , _func_);
			return false;
		}
		
		//conv []interface{} = {procid , proc_name}		
		v_list , ok := pconfig.FileConfig.ClientList[i].([]interface{});
		if !ok {
			log.Err("%s not illegal! v:%v" , _func_ , pconfig.FileConfig.ClientList[i]);
			return false;
		}
				
		//parse		
		  //proc id		
		if proc_id , ok := v_list[0].(float64); ok {
		    pwatch.ProcId = int(proc_id);
		} else {
			log.Err("%s proc_id not valid! v:%v" , _func_ , pconfig.FileConfig.ClientList[i]);
			//return false;
		}
		
		  //proc name
		if proc_name , ok := v_list[1].(string); ok {
			pwatch.ProcName = proc_name;
		} else {
			log.Err("%s proc_name not valid! v:%v" , _func_ , pconfig.FileConfig.ClientList[i]);
		}
		
		//set
		pconfig.WatchMap[pwatch.ProcId] = pwatch;
		log.Info("%s [%d] %v --> %v" , _func_ , i , v_list , *pwatch);		
	}
	
	//return
	log.Info("%s finish! map:%v" , _func_ , pconfig.WatchMap);
	return true;
}


func StartRecver(pconfig *Config) *ReportRecver {
	var _func_ = "<StartRecver>";
	log := pconfig.Comm.Log;
	
	//new
	precver := new(ReportRecver);
	if precver == nil {
		log.Err("%s failed! new recver fail!" , _func_);
		return nil;
	}
	
	//resolve addr
	listen_addr , err := net.ResolveUDPAddr("udp", pconfig.FileConfig.ListenAddr);
	if err != nil {
		log.Err("%s failed! resolve addr:%s failed! err:%v" , _func_ , pconfig.FileConfig.ListenAddr , err);
		return nil;
	}
	
	//listened
	conn , err := net.ListenUDP("udp", listen_addr);
	if err != nil {
		log.Err("%s listen failed at %s. err:%v" , _func_ , pconfig.FileConfig.ListenAddr , err);
		return nil;
	}
	
	//set info
	precver.config = pconfig;
	precver.local_addr = pconfig.FileConfig.ListenAddr;
	precver.recv_conn = conn;
	precver.exit_ch = make(chan bool , 1);
	
	//recv
	log.Info("%s at %s success!" , _func_ , pconfig.FileConfig.ListenAddr);
	precver.serve();
	return precver;
}

func (precver *ReportRecver) Close() {
	precver.exit_ch <- true;
}


/*-------------------------------STATIC FUNC-------------------------------*/
func (precver *ReportRecver) serve() {
	go func() {
		var _func_ = "<recver.serve>";
		pconfig := precver.config;
		log := pconfig.Comm.Log;
		
		for {
			time.Sleep(10 * time.Millisecond);
		    //check exit
		    if len(precver.exit_ch) > 0 {
		    	log.Info("%s exit" , _func_);
		    	break;
		    }
		
		    //recv
		    precver.recv();
		
		}
	}();
}

//recv report
func (precver *ReportRecver) recv() {
	var _func_ = "<recver.recv>";
	log := precver.config.Comm.Log;


    //recv    
	var buff = make([]byte , comm.MAX_REPORT_MSG_LEN);
	var conn = precver.recv_conn;
	for i:=0; i<MAX_RECV_PER_TICK; i++ {
    	buff = buff[:cap(buff)];
    	//set dead line 
		conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond)); //set timeout
		
    	//recv
    	n , peer_addr , err := conn.ReadFromUDP(buff);
    	if err != nil {
    	    break;
    	}
    	if n == 0 {
    	    log.Debug("%s addr %s no pkg!" , _func_ , precver.local_addr);
    	    break;
    	}
    	
    	//decode
    	pmsg := new(comm.ReportMsg);
    	err = comm.DecodeReportMsg(buff[:n], pmsg);
    	if err != nil {
    	    log.Err("%s decode from %s failed! err:%v" , _func_ ,  peer_addr.String() , err);
   		    continue;
    	}
    	
    	//append
    	//log.Debug("%s from:%s recv proc:%d proto:%d iv:%d sv:%s" , _func_ , peer_addr.String() , pmsg.ProcId , pmsg.ProtoId , pmsg.IntValue , pmsg.StrValue);
    	precver.handle_msg(pmsg, peer_addr);
	}
}

func (precver *ReportRecver) handle_msg(pmsg *comm.ReportMsg , peer_addr *net.UDPAddr) {
	var _func_ = "<recver.handle_msg>";
	pconfig := precver.config;
	log := pconfig.Comm.Log;
	
	//get procid
	proc_id := pmsg.ProcId;
	pwatch , ok := pconfig.WatchMap[proc_id];
	if !ok {
		log.Err("%s no watcher %d" , _func_ , proc_id);
		return;
	}
	
	
	//set
	pstat := &pwatch.Stat;
	pstat.addr = peer_addr;
	switch pmsg.ProtoId {
		case comm.REPORT_PROTO_SERVER_START: //report serverstart
		    log.Debug("%s <%d:%s> starts:%v" , _func_ , pwatch.ProcId , pwatch.ProcName , pmsg.IntValue);
	        pstat.StartTime = time.Unix(pmsg.IntValue , 0);
	    case comm.REPORT_PROTO_SERVER_HEART: //report hearbeat
	        //log.Debug("%s <%d:%s> heart:%v" , _func_ , pwatch.ProcId , pwatch.ProcName , pmsg.IntValue);
	        pstat.HeartBeat = time.Unix(pmsg.IntValue , 0);
	    case comm.REPORT_PROTO_CONN_NUM: //report connection number
	        //log.Debug("%s <%d:%s> conn_num:%v str:%s" , _func_ , pwatch.ProcId , pwatch.ProcName , pmsg.IntValue , pmsg.StrValue);
	        pstat.ConnNum = int(pmsg.IntValue);
	        pstat.ConnStr = pmsg.StrValue;
	    case comm.REPORT_PROTO_SYNC_SERVER: //sync server
	        log.Debug("%s <%d:%s> sync_server" , _func_ , pwatch.ProcId , pwatch.ProcName);
	        psync , ok := pmsg.Sub.(*comm.SyncServerMsg);
	        if !ok {
	        	log.Err("%s <%d:%s> sync_server decode sync_msg failed!" , _func_ , pwatch.ProcId , pwatch.ProcName);
	        	break;
	        }
	        pstat.StartTime = time.Unix(psync.StartTime , 0);
	        
	                
	    default:
	        log.Err("%s unknown proto:%d" , _func_ , pmsg.ProtoId);        	    
	}
	
		
	return;
}


