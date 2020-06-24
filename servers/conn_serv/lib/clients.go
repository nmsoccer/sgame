package lib

import (
  "time"
  "sgame/servers/comm"
  "sgame/proto/cs"
)

func ReadClients(pconfig *Config) int64{
	var _func_ = "<ReadClients>";
	log := pconfig.Comm.Log;
		
	//get results
	var results []*comm.ClientPkg = pconfig.TcpServ.Recv(pconfig.Comm); //comm.*ClientPkg
	
	//log.Debug("%s get results:%v len:%d" , _func_ , results , len(results));
	if results == nil || len(results)==0 {
		return 0;
	}
	
	start_ts := time.Now().UnixNano();	
	//print
	for i:=0; i<len(results); i++ {		
		log.Debug("%s key:%v , type:%d , read:%v" , _func_ , results[i].ClientKey , results[i].PkgType ,  results[i].Data);
		HandleClientPkg(pconfig, results[i]);
	}
	
	//diff
	diff := time.Now().UnixNano()-start_ts;
	log.Debug("%s cost %dus pkg:%d" , _func_ , diff/1000 , len(results));
	return diff;
}


func HandleClientPkg(pconfig *Config , pclient *comm.ClientPkg) {
	var _func_ = "<HandleClientPkg>";
	var gmsg cs.GeneralMsg;
	log := pconfig.Comm.Log;
	
	//decode msg
	err := cs.DecodeMsg(pclient.Data, &gmsg);
	if err != nil {
		log.Err("%s decode msg failed! err:%v" , _func_ , err);
		return;
	}
	
	proto_id := gmsg.ProtoId;
	var conv_err = true;
	//convert
	switch proto_id {
		case cs.CS_PROTO_PING_REQ:
		    pmsg , ok := gmsg.SubMsg.(*cs.CSPingReq);
		    if ok {
		        log.Debug("%s recv proto:%d success! v:%v" , _func_ , proto_id , *pmsg);
		        SendPingReq(pconfig, pclient.ClientKey , pmsg);
		        conv_err = false;
		    }
		case cs.CS_PROTO_LOGIN_REQ:
		    pmsg , ok := gmsg.SubMsg.(*cs.CSLoginReq);
		    if ok {
		    	log.Debug("%s recv proto:%d success! v:%v" , _func_ , proto_id , *pmsg);
		    	SendLoginReq(pconfig, pclient.ClientKey , pmsg);
		    	conv_err = false;
		    }  
		default:
		    log.Err("%s illegal proto:%d" , _func_ , proto_id);
		    return;   
	}
	
	//log
	if conv_err {
	    log.Err("%s conv proto:%d failed!" , _func_ , proto_id);
	}			
} 

func SendToClient(pconfig *Config , client_key int64 , gmsg *cs.GeneralMsg) bool{
	var _func_ = "<SendToClient>";
	log := pconfig.Comm.Log;
	
	//enc msg
	enc_data , err := cs.EncodeMsg(gmsg);
	if err != nil {
		log.Err("%s encode msg failed! key:%v err:%v" , _func_ , client_key , err);
		return false;
	}
	
	//make pkg
	pclient := new(comm.ClientPkg);
	pclient.ClientKey = client_key;
	pclient.Data = enc_data;
	
	//Send
	ret := pconfig.TcpServ.Send(pconfig.Comm , pclient);
	log.Debug("%s send to client ret:%d data:%v" , _func_ , ret , pclient.Data);
	return true;	
}
