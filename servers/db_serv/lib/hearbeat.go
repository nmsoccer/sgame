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
	pconfig, ok := arg.(*Config)
	if !ok {
		return
	}
	log := pconfig.Comm.Log

	//ss_msg
	var ss_msg ss.SSMsg
	pheart := new(ss.MsgHeartBeatReq)
	pheart.Ts = curr_ts
	err := comm.FillSSPkg(&ss_msg , ss.SS_PROTO_TYPE_HEART_BEAT_REQ , pheart)
	if err != nil {
		log.Err("%s gen ss_msg failed! err:%v" , _func_ , err)
	} else {
		//send msg
		for _, target_id := range pconfig.FileConfig.TargetServs {
			SendToServ(pconfig , target_id , &ss_msg)
		}
	}


	//report
	pconfig.ReportServ.Report(comm.REPORT_PROTO_SERVER_HEART, curr_ts, "", nil)
	pconfig.ReportServ.Report(comm.REPORT_PROTO_CONN_NUM, int64(CalcRedisConn(pconfig)), "RedisConn", nil)
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
	pclient := SelectRedisClient(pconfig , REDIS_OPT_W)

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
