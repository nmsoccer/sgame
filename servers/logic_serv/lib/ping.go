package lib

import (
    "sgame/proto/ss"
)

func RecvPingReq(pconfig *Config , preq *ss.MsgPingReq , from int) {
	var _func_ = "<RecvPingReq>";
	log := pconfig.Comm.Log;
	
	log.Debug("%s get ping req! client_key:%v ts:%v from:%d" , _func_ , preq.ClientKey , preq.Ts , from);	
	//Back
	var  rsp ss.SSMsg;
	rsp.ProtoType = ss.SS_PROTO_TYPE_PING_RSP;
	    //body
	pbody := new(ss.SSMsg_PingRsp);
	pbody.PingRsp = new(ss.MsgPingRsp);
	pbody.PingRsp.ClientKey = preq.ClientKey;
	pbody.PingRsp.Ts = preq.Ts;
	rsp.MsgBody = pbody;
	
	//encode
	buff , err := ss.Pack(&rsp);
	if err != nil {
		log.Err("%s pack rsp failed! err:%v key:%v" , _func_ , err , preq.ClientKey);
		return;
	}
	
	//sendback
	ok := SendToConnect(pconfig, buff);
	if !ok {
		log.Err("%s send back failed! key:%v" , _func_ , preq.ClientKey);
		return;
	}
	log.Debug("%s send back success! key:%v" , _func_ , preq.ClientKey);
	return;	
}

