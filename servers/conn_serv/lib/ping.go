package lib

import (
    "sgame/proto/ss"
    "sgame/proto/cs"
)


func SendPingReq(pconfig *Config , client_key int64 , pmsg *cs.CSPingReq) {
	var _func_ = "<SendPingReq>";
	log := pconfig.Comm.Log;
	
	//init pkg
	var ss_req ss.SSMsg;
	  //proto
	ss_req.ProtoType = ss.SS_PROTO_TYPE_PING_REQ;
	  //body
	sp := new(ss.SSMsg_PingReq);
	sp.PingReq = new(ss.MsgPingReq);
	sp.PingReq.ClientKey = client_key;
	sp.PingReq.Ts = pmsg.TimeStamp;
	  //finish
	ss_req.MsgBody = sp;
	
	//pack
	buff , err := ss.Pack(&ss_req);
	if err != nil {
		log.Err("%s pack failed! client:%v err:%v" , _func_ , client_key , err);
		return;
	}
	
	//send msg	
    //ret := pconfig.Comm.Proc.Send(pconfig.FileConfig.LogicServ, buff , len(buff));
    ok := SendToLogic(pconfig, buff);
    if !ok {
	    log.Err("%s send msg  failed!" , _func_);
    }
    log.Debug("%s send success! client_key:%v ts:%v" , _func_  , client_key , pmsg.TimeStamp);        
	return;
}


func RecvPingRsp(pconfig *Config , pmsg *ss.MsgPingRsp) {
	var _func_ = "<RecvPingRsp>";
	log := pconfig.Comm.Log;	
	log.Debug("%s client:%v ts:%v" , _func_ , pmsg.ClientKey , pmsg.Ts);
		
	//gmsg
	var gmsg cs.GeneralMsg;
	gmsg.ProtoId = cs.CS_PROTO_PING_RSP;
	psub := new(cs.CSPingRsp);
	psub.TimeStamp = pmsg.Ts;
	gmsg.SubMsg = psub;
	
	//send
	SendToClient(pconfig, pmsg.ClientKey , &gmsg);	
}
