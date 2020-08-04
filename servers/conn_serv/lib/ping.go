package lib

import (
	"sgame/proto/cs"
	"sgame/proto/ss"
)

func SendPingReq(pconfig *Config, client_key int64, pmsg *cs.CSPingReq) {
	var _func_ = "<SendPingReq>"
	log := pconfig.Comm.Log

	//init pkg
	var ss_req ss.SSMsg
	//proto
	ss_req.ProtoType = ss.SS_PROTO_TYPE_PING_REQ
	//body
	sp := new(ss.SSMsg_PingReq)
	sp.PingReq = new(ss.MsgPingReq)
	sp.PingReq.ClientKey = client_key
	sp.PingReq.Ts = pmsg.TimeStamp
	//finish
	ss_req.MsgBody = sp

	//pack
	buff, err := ss.Pack(&ss_req)
	if err != nil {
		log.Err("%s pack failed! client:%v err:%v", _func_, client_key, err)
		return
	}

	//send msg
	//ret := pconfig.Comm.Proc.Send(pconfig.FileConfig.LogicServ, buff , len(buff));
	ok := SendToLogic(pconfig, buff)
	if !ok {
		log.Err("%s send msg  failed!", _func_)
	}
	log.Debug("%s send success! client_key:%v ts:%v", _func_, client_key, pmsg.TimeStamp)
	return
}

func RecvPingRsp(pconfig *Config, prsp *ss.MsgPingRsp) {
	var _func_ = "<RecvPingRsp>"
	log := pconfig.Comm.Log
	log.Debug("%s client:%v ts:%v", _func_, prsp.ClientKey, prsp.Ts)

	//response
	var pmsg *cs.CSPingRsp
	pv, err := cs.Proto2Msg(cs.CS_PROTO_PING_RSP)
	if err != nil {
		log.Err("%s proto2msg failed! proto:%d err:%v", _func_, cs.CS_PROTO_PING_RSP, err)
		return
	}
	pmsg, ok := pv.(*cs.CSPingRsp)
	if !ok {
		log.Err("%s proto2msg type illegal!  proto:%d", _func_, cs.CS_PROTO_PING_RSP)
		return
	}
	pmsg.TimeStamp = prsp.Ts

	//send
	SendToClient(pconfig, prsp.ClientKey, cs.CS_PROTO_PING_RSP, pmsg)
}
