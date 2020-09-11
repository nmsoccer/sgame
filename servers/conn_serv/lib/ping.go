package lib

import (
	"sgame/proto/cs"
	"sgame/proto/ss"
	"sgame/servers/comm"
)

func SendPingReq(pconfig *Config, client_key int64, pmsg *cs.CSPingReq) {
	var _func_ = "<SendPingReq>"
	log := pconfig.Comm.Log

	//create pkg
	var ss_msg ss.SSMsg
	pPingReq := new(ss.MsgPingReq)
	pPingReq.ClientKey = client_key
	pPingReq.Ts = pmsg.TimeStamp

	//FILL
	err := comm.FillSSPkg(&ss_msg , ss.SS_PROTO_TYPE_PING_REQ , pPingReq)
	if err != nil {
		log.Err("%s fill ss failed! client:%v err:%v", _func_, client_key, err)
		return
	}

	//send msg
	ok := SendToLogic(pconfig, &ss_msg)
	if !ok {
		log.Err("%s send msg  failed!", _func_)
	}
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
