package lib

import (
    "sgame/proto/ss"
    "sgame/proto/cs"
    "sgame/servers/comm"    
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
    ret := pconfig.Comm.Proc.Send(pconfig.FileConfig.LogicServ, buff , len(buff));
    if ret < 0 {
	    log.Err("%s send msg to %d failed! err:%d" , _func_ , pconfig.FileConfig.LogicServ , ret);
    }
    log.Debug("%s send to %d success! client_key:%v ts:%v" , _func_ , pconfig.FileConfig.LogicServ , client_key , pmsg.TimeStamp);        
	return;
}


func RecvPingRspMsg(pconfig *Config , pmsg *ss.MsgPingRsp) {
	var _func_ = "<RecvPingRspMsg>";
	log := pconfig.Comm.Log;	
	log.Debug("%s client:%v ts:%v" , _func_ , pmsg.ClientKey , pmsg.Ts);
		
	  //pack
	var gmsg cs.GeneralMsg;
	gmsg.ProtoId = cs.CS_PROTO_PING_RSP;
	psub := new(cs.CSPingRsp);
	psub.TimeStamp = pmsg.Ts;
	gmsg.SubMsg = psub;
	
	  //encode	  
	enc_data , err := cs.EncodeMsg(&gmsg);
	if err != nil {
		log.Err("%s encode msg failed! key:%v err:%v" , _func_ , pmsg.ClientKey , err);
		return;
	}
	
	//To Client
	pclient := new(comm.ClientPkg);
	pclient.ClientKey = pmsg.ClientKey;
	pclient.Data = enc_data;
	
	//Send
	ret := pconfig.TcpServ.Send(pconfig.Comm , pclient);
	log.Debug("%s send to client ret:%d data:%v" , _func_ , ret , pclient.Data);
	return;
}
