package lib

import (
	"html/template"
	"net"
	"sgame/servers/comm"
	"time"
)

const (
	MAX_RECV_PER_TICK = 10
    CMD_CHAN_SIZE = 10000
    MAX_SND_CMD_TICK = 100

)

type PeerStat struct {
	addr      *net.UDPAddr
	StartTime time.Time
	HeartBeat time.Time
	StopTime  time.Time
	ConnStr   string
	ConnNum   int
	CmdTime  time.Time //latest cmd time
	CmdStat string //refer CMD_STAT_XX
	CmdInfo  string
	MonitorInfo template.HTML //monitor info
}


type ReportRecver struct {
	config     *Config
	local_addr string
	recv_conn  *net.UDPConn
	exit_ch    chan bool
	cmd_ch chan *comm.ReportMsg  //manager to other
}

//parse FileConfig.ClientList --> Config.WatchMap
func ParseClientList(pconfig *Config) bool {
	var _func_ = "ParseClientList"
	log := pconfig.Comm.Log

	//init
	log.Info("%s client list:%v", _func_, pconfig.FileConfig.ClientList)
	pconfig.WatchMap = make(map[int]*WatchClient)
	pconfig.Name2Id = make(map[string]int)
	//psarse
	for i := 0; i < len(pconfig.FileConfig.ClientList); i++ {
		//new
		pwatch := new(WatchClient)
		if pwatch == nil {
			log.Err("%s new client failed!", _func_)
			return false
		}

		//conv []interface{} = {procid , proc_name}
		v_list, ok := pconfig.FileConfig.ClientList[i].([]interface{})
		if !ok {
			log.Err("%s not illegal! v:%v", _func_, pconfig.FileConfig.ClientList[i])
			return false
		}

		//parse
		//proc id
		if proc_id, ok := v_list[0].(float64); ok {
			pwatch.ProcId = int(proc_id)
		} else {
			log.Err("%s proc_id not valid! v:%v", _func_, pconfig.FileConfig.ClientList[i])
			//return false;
		}

		//proc name
		if proc_name, ok := v_list[1].(string); ok {
			pwatch.ProcName = proc_name
		} else {
			log.Err("%s proc_name not valid! v:%v", _func_, pconfig.FileConfig.ClientList[i])
		}

		//set
		pconfig.WatchMap[pwatch.ProcId] = pwatch
		pconfig.Name2Id[pwatch.ProcName] = pwatch.ProcId
		log.Info("%s [%d] %v --> %v", _func_, i, v_list, *pwatch)
	}

	//return
	log.Info("%s finish! map:%v", _func_, pconfig.WatchMap)
	return true
}

func StartRecver(pconfig *Config) *ReportRecver {
	var _func_ = "<StartRecver>"
	log := pconfig.Comm.Log

	//new
	precver := new(ReportRecver)
	if precver == nil {
		log.Err("%s failed! new recver fail!", _func_)
		return nil
	}

	//resolve addr
	listen_addr, err := net.ResolveUDPAddr("udp", pconfig.FileConfig.ListenAddr)
	if err != nil {
		log.Err("%s failed! resolve addr:%s failed! err:%v", _func_, pconfig.FileConfig.ListenAddr, err)
		return nil
	}

	//listened
	conn, err := net.ListenUDP("udp", listen_addr)
	if err != nil {
		log.Err("%s listen failed at %s. err:%v", _func_, pconfig.FileConfig.ListenAddr, err)
		return nil
	}

	//set info
	precver.config = pconfig
	precver.local_addr = pconfig.FileConfig.ListenAddr
	precver.recv_conn = conn
	precver.exit_ch = make(chan bool, 1)
    precver.cmd_ch = make(chan *comm.ReportMsg , CMD_CHAN_SIZE);
	//recv
	log.Info("%s at %s success!", _func_, pconfig.FileConfig.ListenAddr)
	precver.serve()
	return precver
}

func (precver *ReportRecver) Close() {
	precver.exit_ch <- true
}

/*-------------------------------STATIC FUNC-------------------------------*/
func (precver *ReportRecver) serve() {
	go func() {
		var _func_ = "<recver.serve>"
		pconfig := precver.config
		log := pconfig.Comm.Log

		for {
			time.Sleep(10 * time.Millisecond)
			//check exit
			if len(precver.exit_ch) > 0 {
				log.Info("%s exit", _func_)
				break
			}

			//recv
			precver.recv()

			//send
			precver.send();

		}
	}()
}

//recv report
func (precver *ReportRecver) recv() {
	var _func_ = "<recver.recv>"
	log := precver.config.Comm.Log

	//recv
	var buff = make([]byte, comm.MAX_REPORT_MSG_LEN)
	var conn = precver.recv_conn
	for i := 0; i < MAX_RECV_PER_TICK; i++ {
		buff = buff[:cap(buff)]
		//set dead line
		conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond)) //set timeout

		//recv
		n, peer_addr, err := conn.ReadFromUDP(buff)
		if err != nil {
			break
		}
		if n == 0 {
			log.Debug("%s addr %s no pkg!", _func_, precver.local_addr)
			break
		}

		//decode
		pmsg := new(comm.ReportMsg)
		err = comm.DecodeReportMsg(buff[:n], pmsg)
		if err != nil {
			log.Err("%s decode from %s failed! err:%v", _func_, peer_addr.String(), err)
			continue
		}

		//append
		//log.Debug("%s from:%s recv proc:%d proto:%d iv:%d sv:%s" , _func_ , peer_addr.String() , pmsg.ProcId , pmsg.ProtoId , pmsg.IntValue , pmsg.StrValue);
		precver.handle_msg(pmsg, peer_addr)
	}
}

//send
func (pserv *ReportRecver) send() {
	var _func_ = "<recver.send>"
	pconfig := pserv.config;
	log := pconfig.Comm.Log;

	//check len
	snd_num := len(pserv.cmd_ch);
    if snd_num > MAX_SND_CMD_TICK {
    	snd_num = MAX_SND_CMD_TICK;
	}
    if snd_num <= 0 {
    	return;
	}

    //send pkg
    var pmsg *comm.ReportMsg;
    for i:=0; i<snd_num; i++ {
    	pmsg = <- pserv.cmd_ch;
    	log.Debug("%s target:%d msg:%v" , _func_ , pmsg.ProcId , *pmsg);

    	//watcher
    	pconfig.watch_lock.RLock();
    	pw := pconfig.WatchMap[pmsg.ProcId];
    	if pw == nil {
    		log.Err("%s proc_id illegal! proc:%d proto:%d" , _func_ , pmsg.ProcId , pmsg.ProtoId);
    		continue;
		}
		pconfig.watch_lock.RUnlock();

		//check addr
    	if pw.Stat.addr == nil {
    		log.Err("%s send cmd to %s fail! addr nil! proto:%d" , _func_ , pw.ProcName , pmsg.ProtoId);
    		continue;
		}

		//enc data
		enc_data , err := comm.EncodeReportMsg(pmsg);
		if err != nil {
			log.Err("%s encode failed! target:%s proto:%d" , _func_ , pw.ProcName , pmsg.ProtoId);
			continue;
		}

		//send
		pserv.recv_conn.WriteTo(enc_data , pw.Stat.addr);
	}
}


func (precver *ReportRecver) handle_msg(pmsg *comm.ReportMsg, peer_addr *net.UDPAddr) {
	var _func_ = "<recver.handle_msg>"
	pconfig := precver.config
	log := pconfig.Comm.Log

	//get procid
	proc_id := pmsg.ProcId
	pconfig.watch_lock.RLock()
	pwatch, ok := pconfig.WatchMap[proc_id]
	if !ok {
		log.Err("%s no watcher %d", _func_, proc_id)
		return
	}
	pconfig.watch_lock.RUnlock()

	//set
	pstat := &pwatch.Stat
	pstat.addr = peer_addr
	switch pmsg.ProtoId {
	case comm.REPORT_PROTO_SERVER_START: //report serverstart
		log.Debug("%s <%d:%s> starts:%v", _func_, pwatch.ProcId, pwatch.ProcName, pmsg.IntValue)
		pstat.StartTime = time.Unix(pmsg.IntValue, 0)
		if pstat.StartTime.After(pstat.StopTime) { //new start clear xx
			pstat.StopTime = time.Unix(0, 0)
			pstat.CmdTime = time.Unix(0 , 0)
			pstat.CmdStat = comm.CMD_STAT_NONE
			pstat.CmdInfo = comm.CMD_CMD_NONE
		}
	case comm.REPORT_PROTO_SERVER_HEART: //report hearbeat
		//log.Debug("%s <%d:%s> heart:%v" , _func_ , pwatch.ProcId , pwatch.ProcName , pmsg.IntValue);
		pstat.HeartBeat = time.Unix(pmsg.IntValue, 0)
	case comm.REPORT_PROTO_CONN_NUM: //report connection number
		//log.Debug("%s <%d:%s> conn_num:%v str:%s" , _func_ , pwatch.ProcId , pwatch.ProcName , pmsg.IntValue , pmsg.StrValue);
		pstat.ConnNum = int(pmsg.IntValue)
		pstat.ConnStr = pmsg.StrValue
	case comm.REPORT_PROTO_SYNC_SERVER: //sync server
		//log.Debug("%s <%d:%s> sync_server", _func_, pwatch.ProcId, pwatch.ProcName)
		psync, ok := pmsg.Sub.(*comm.SyncServerMsg)
		if !ok {
			log.Err("%s <%d:%s> sync_server decode sync_msg failed!", _func_, pwatch.ProcId, pwatch.ProcName)
			break
		}
		pstat.StartTime = time.Unix(psync.StartTime, 0)
		if pstat.StartTime.After(pstat.StopTime) { //new start clear old stop
			pstat.StopTime = time.Unix(0, 0)
		}
	case comm.REPORT_PROTO_SERVER_STOP: //report server stop time
		log.Info("%s <%d:%s> stop:%v", _func_, pwatch.ProcId, pwatch.ProcName, pmsg.IntValue)
		pstat.StopTime = time.Unix(pmsg.IntValue, 0)
	case comm.REPORT_PROTO_CMD_RSP: //report from cmd
	    log.Info("%s <%d:%s> cmd_rsp. time:%d ret:%s" , _func_ , pwatch.ProcId , pwatch.ProcName , pmsg.IntValue ,
	    	pmsg.StrValue);
	    //check time ts must match
	    if pwatch.Stat.CmdTime.Unix() == pmsg.IntValue {
	    	log.Info("cmd time matched");
	    	pwatch.Stat.CmdStat = pmsg.StrValue;
		} else {
			log.Err("%s cmd ts not match! %d vs %d" , _func_ , pwatch.Stat.CmdTime.Unix() , pmsg.IntValue);
		}

	case comm.REPORT_PROTO_MONITOR:
		//log.Debug("%s <%d:%s> monitor:%s", _func_, pwatch.ProcId, pwatch.ProcName, pmsg.StrValue);
		//str := strings.Replace(pmsg.StrValue , "\n" , "<br/>" , -1);
		//str = strings.Replace(str , " " , "&nbsp&nbsp&nbsp&nbsp" , -1);
		pwatch.Stat.MonitorInfo = template.HTML(pmsg.StrValue);


	default:
		log.Err("%s unknown proto:%d", _func_, pmsg.ProtoId)
	}

	return
}
