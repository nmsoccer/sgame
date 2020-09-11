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
	log := pconfig.Comm.Log;
	
	//proto
    var ss_msg ss.SSMsg;
    pHeartBeatReq := new(ss.MsgHeartBeatReq);
    pHeartBeatReq.Ts = time.Now().Unix();


    //gen
    err := comm.FillSSPkg(&ss_msg , ss.SS_PROTO_TYPE_HEART_BEAT_REQ , pHeartBeatReq);
    if err != nil {
    	log.Err("%s gen ss failed! err:%v" , _func_ , err);
    } else {
		for _ , logic_id := range pconfig.FileConfig.LogicServList {
			SendToServ(pconfig , logic_id , &ss_msg)
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
