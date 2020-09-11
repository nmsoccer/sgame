package lib

import (
    "sgame/proto/ss"
	"sgame/servers/comm"
)

func RecvPingReq(pconfig *Config , preq *ss.MsgPingReq , from int) {
	var _func_ = "<RecvPingReq>";
	log := pconfig.Comm.Log;
	
	log.Debug("%s get ping req! client_key:%v ts:%v from:%d" , _func_ , preq.ClientKey , preq.Ts , from);	
	//Back
	var  ss_msg ss.SSMsg;
	pPingRsp := new(ss.MsgPingRsp);
	pPingRsp.ClientKey = preq.ClientKey;
	pPingRsp.Ts = preq.Ts;

	
	//encode
	err := comm.FillSSPkg(&ss_msg , ss.SS_PROTO_TYPE_PING_RSP , pPingRsp);
	if err != nil {
		log.Err("%s gen ss failed! err:%v key:%v" , _func_ , err , preq.ClientKey);
		return;
	}
	
	//sendback
	ok := SendToConnect(pconfig, &ss_msg);
	if !ok {
		log.Err("%s send back failed! key:%v" , _func_ , preq.ClientKey);
		return;
	}
	log.Debug("%s send back success! key:%v" , _func_ , preq.ClientKey);
	return;	
}

