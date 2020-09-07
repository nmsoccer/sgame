package lib

import (
	"sgame/proto/ss"
	"sgame/servers/comm"
	"time"
	//"math/rand"
)

func SendHeartBeatMsg(arg interface{}) {
	var pconfig *Config
	var _func_ = "<SendHeartBeatMsg>"
	curr_ts := time.Now().Unix()
	/*

		if ((curr_ts-last_send) < heart_beat_circle) {
			return;
		}

		//go>>>
		last_send = curr_ts;
	*/
	pconfig, ok := arg.(*Config)
	if !ok {
		return
	}
	lp := pconfig.Comm.Log

	//proto
	var ss_req ss.SSMsg
	ss_req.ProtoType = ss.SS_PROTO_TYPE_HEART_BEAT_REQ
	hb := new(ss.SSMsg_HeartBeatReq)
	hb.HeartBeatReq = new(ss.MsgHeartBeatReq)
	hb.HeartBeatReq.Ts = time.Now().Unix()
	ss_req.MsgBody = hb

	//pack
	buff, err := ss.Pack(&ss_req)
	if err != nil {
		lp.Err("%s pack failed! err:%v", _func_, err)
		return
	} else {
		//lp.Debug("%s pack success! buff:%v len:%d cap:%d" , _func_ , buff , len(buff) , cap(buff));
	}

	proc := pconfig.Comm.Proc
	//send msg
	for _, target_id := range pconfig.FileConfig.TargetServs {
		ret := proc.SendByLock(target_id, buff, len(buff)) //db_server should use lock by sending proc
		if ret < 0 {
			lp.Err("send msg to %d failed! err:%d", target_id, ret)
		}
	}

	//report
	pconfig.ReportServ.Report(comm.REPORT_PROTO_SERVER_HEART, curr_ts, "", nil)
	pconfig.ReportServ.Report(comm.REPORT_PROTO_CONN_NUM, int64(pconfig.RedisClient.GetConnNum()), "RedisConn", nil)
	return
}

func RecvHeartBeatReq(pconfig *Config, preq *ss.MsgHeartBeatReq, from int) {
	var _func_ = "<RecvHeartBeatReq>"
	var log = pconfig.Comm.Log
	var stats = pconfig.Comm.PeerStats

	_, ok := stats[from]
	if ok {
		//log.Debug("%s update heartbeat server:%d %v --> %v" , _func_ , from , last , preq.GetTs());
	} else {
		log.Debug("%s set  heartbeat server:%d %v", _func_, from, preq.GetTs())
	}
	stats[from] = preq.GetTs()
}

//call back of heartbeat to redis
func cb_heartbeat_redis(pconfig *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_heartbeat_redis>"
	log := pconfig.Log

	//conv ret
	ret, err := comm.Conv2String(result)
	if err != nil {
		log.Err("%s conv2int failed! result:%v err:%v ret:%v", _func_, result, err, ret)
		return
	}

	//get arg
	_, ok := cb_arg[0].(int64)
	if ok {
		//log.Debug("%s done! result:%v ts:%d" , _func_ , ret , ts);
	}
}

func HeartBeatToRedis(arg interface{}) {
	var pconfig *Config
	pconfig, ok := arg.(*Config)
	if !ok {
		return
	}
	curr_ts := time.Now().Unix()
	pclient := pconfig.RedisClient

	//this is a press test change heart_beat_circle=1
	/*
		test_num := rand.Intn(pconfig.FileConfig.NormalConn*1000);
		pconfig.Comm.Log.Debug("%s will send %d reqs", "<HeartBeatToRedis>" , test_num);*/
	test_num := 1
	for i := 0; i < test_num; i++ {
		pclient.RedisExeCmd(pconfig.Comm, cb_heartbeat_redis, []interface{}{curr_ts}, "SET", "HEARTBEAT", curr_ts)
	}
}

func ReportSyncServer(arg interface{}) {
	var pconfig *Config
	pconfig, ok := arg.(*Config)
	if !ok {
		return
	}

	//msg
	pmsg := new(comm.SyncServerMsg)
	pmsg.StartTime = pconfig.Comm.StartTs

	//send
	pconfig.ReportServ.Report(comm.REPORT_PROTO_SYNC_SERVER, 0, "", pmsg)
}
