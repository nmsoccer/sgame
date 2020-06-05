package lib

import (
	"sgame/servers/comm"
    "sgame/proto"
    "time"
    //"math/rand"
)

var last_send int64;
var last_hb_redis int64;
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
    var ss_req proto.SSMsgReq;
    ss_req.ProtoType = proto.SS_PROTO_TYPE_HEART_BEAT_REQ;
    hb := new(proto.SSMsgReq_HeartBeat);
    hb.HeartBeat = new(proto.MsgHeartBeatReq);
    hb.HeartBeat.Ts = time.Now().Unix();
    ss_req.MsgBody = hb;

    //pack
    buff , err := proto.Pack(&ss_req);
    if err != nil {
    	lp.Err("%s pack failed! err:%v" , _func_ , err);
    	return;
    } else {
    	//lp.Debug("%s pack success! buff:%v len:%d cap:%d" , _func_ , buff , len(buff) , cap(buff));
    }
    
    
    proc := pconfig.Comm.Proc;
    //send msg
    ret := proc.Send(pconfig.FileConfig.LogicServs[0] , buff , len(buff));
    if ret < 0 {
	    lp.Err("send msg to %d failed! err:%d" , pconfig.FileConfig.LogicServs[0] , ret);
    }
    
	return;
}


func RecvHeartBeatReq(pconfig *Config , preq *proto.MsgHeartBeatReq , from int) {
	var _func_ = "<RecvHeartBeatReq>";
	var log = pconfig.Comm.Log;
	var stats = pconfig.Comm.PeerStats;
	
	last , ok := stats[from];
	if ok {
	    log.Debug("%s update heartbeat server:%d %v --> %v" , _func_ , from , last , preq.GetTs());	
	} else {
		log.Debug("%s set  heartbeat server:%d %v" , _func_ , from , preq.GetTs());	
	}
	stats[from] = preq.GetTs();
}

//call back of heartbeat to redis
func cb_heartbeat_redis(pconfig *comm.CommConfig , result interface{} , cb_arg []interface{}) {
	var _func_ = "<cb_heartbeat_redis>";
	log := pconfig.Log;
	
	//conv ret
	ret , err := comm.Conv2String(result);
	if err != nil {
		log.Err("%s conv2int failed! result:%v err:%v ret:%v" , _func_ , result , err , ret);
		return;
	}
	
	//get arg
	ts , ok := cb_arg[0].(int64);
	if ok {
	    log.Debug("%s done! result:%v ts:%d" , _func_ , ret , ts);
	}	
}


func HeartBeatToRedis(pconfig *Config) {
	curr_ts := time.Now().Unix();
	if curr_ts < last_hb_redis + heart_beat_circle {
		return;
	}
	last_hb_redis = curr_ts;
	if pconfig.FileConfig.RedisOpen != 1 {
		return;
	}
	pclient := pconfig.RedisClient;
	
	//this is a press test change heart_beat_circle=1
	/*
	test_num := rand.Intn(pconfig.FileConfig.NormalConn*1000);
	pconfig.Comm.Log.Debug("%s will send %d reqs", "<HeartBeatToRedis>" , test_num);*/
	test_num := 1;
	for i:=0; i<test_num ; i++ {
	    pclient.RedisExeCmd(pconfig.Comm , cb_heartbeat_redis, []interface{}{curr_ts}, "SET", "HEARTBEAT" , curr_ts);
	}
}


