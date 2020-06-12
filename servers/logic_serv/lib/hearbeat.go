package lib

import (
    "sgame/proto/ss"
    "time"
)

var last_send int64;
const (
    heart_beat_circle = 10;
)

func SendHeartBeatMsg(pconfig *Config) {
	var _func_ = "<SendHeartBeatMsg>";
	curr_ts := time.Now().Unix();
	if ((curr_ts-last_send) < heart_beat_circle) {
		return;
	}
	
	//go>>>
	last_send = curr_ts;	
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
    ret := proc.Send(pconfig.FileConfig.ConnServ , buff , len(buff));
    if ret < 0 {
	    lp.Err("send msg to %d failed! err:%d" , pconfig.FileConfig.ConnServ , ret);
    }
    
    ret = proc.Send(pconfig.FileConfig.MasterDb , buff , len(buff));
    if ret < 0 {
	    lp.Err("send msg to %d failed! err:%d" , pconfig.FileConfig.MasterDb , ret);
    }
    
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


