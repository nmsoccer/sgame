package lib

import (
    "sgame/proto/ss"
    "time"
    "sgame/servers/comm"
)


func SendHeartBeatMsg(arg interface{}) {
	var pconfig *Config;
	var _func_ = "<SendHeartBeatMsg>";
	curr_ts := time.Now().Unix();

	pconfig , ok := arg.(*Config);
	if !ok {
		return;
	}
	lp := pconfig.Comm.Log;
	
	//proto
    var ss_req ss.SSMsg;
    ss_req.ProtoType = ss.SS_PROTO_TYPE_HEART_BEAT_REQ;
    hb := new(ss.SSMsg_HeartBeatReq);
    hb.HeartBeatReq = new(ss.MsgHeartBeatReq); 
    hb.HeartBeatReq.Ts = time.Now().Unix();
    ss_req.MsgBody = hb;

    //pack
    buff , err := ss.Pack(&ss_req);
    if err != nil {
    	lp.Err("%s pack failed! err:%v" , _func_ , err);
    	return;
    } else {
    	//lp.Debug("%s pack success! buff:%v len:%d cap:%d" , _func_ , buff , len(buff) , cap(buff));
    }
    
    
    proc := pconfig.Comm.Proc;
    //send msg
    for _ , logic_id := range pconfig.FileConfig.LogicServList {
		ret := proc.Send(logic_id, buff, len(buff));
		if ret < 0 {
			lp.Err("send msg to %d failed! err:%d", logic_id, ret);
		}
	}
    
    
    //report
    pconfig.ReportServ.Report(comm.REPORT_PROTO_SERVER_HEART , curr_ts , "" , nil);
    //pconfig.ReportServ.Report(comm.REPORT_PROTO_CONN_NUM , int64(pconfig.TcpServ.GetConnNum()) , "ClientConn" , nil);
	return;
}

func RecvHeartBeatReq(pconfig *Config , preq *ss.MsgHeartBeatReq , from int) {
	var _func_ = "<RecvHeartBeatReq>";
	var log = pconfig.Comm.Log;
	var stats = pconfig.Comm.PeerStats;
	
	_ , ok := stats[from];
	if ok {
	    //log.Debug("%s update heartbeat server:%d %v --> %v" , _func_ , from , last , preq.GetTs());	
	} else {
		log.Debug("%s set  heartbeat server:%d %v" , _func_ , from , preq.GetTs());	
	}
	stats[from] = preq.GetTs();
}

func ReportSyncServer(arg interface{}) {
	var pconfig *Config;
	pconfig , ok := arg.(*Config);
	if !ok {
		return;
	}
	
	//msg
	pmsg := new(comm.SyncServerMsg);
	pmsg.StartTime = pconfig.Comm.StartTs;
	
	//send
	pconfig.ReportServ.Report(comm.REPORT_PROTO_SYNC_SERVER , 0 , "" , pmsg);
}
