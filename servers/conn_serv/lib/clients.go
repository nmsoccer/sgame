package lib

import (
	"sgame/proto/ss"
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

	//check pkg type
	if pclient.PkgType == comm.CLIENT_PKG_T_CONN_CLOSED {
		log.Info("%s connection closed! key:%v" , _func_ , pclient.ClientKey);
		//clear map
		uid , ok := pconfig.Ckey2Uid[pclient.ClientKey];
		if ok {
            log.Info("%s is already login. uid:%v notify to upper!" , _func_ , uid);
            SendLogoutReq(pconfig , uid , ss.USER_LOGOUT_REASON_LOGOUT_CONN_CLOSED);
            delete(pconfig.Ckey2Uid , pclient.ClientKey);
            //xxxx
            delete(pconfig.Uid2Ckey , uid);
		} else {
			log.Info("%s no need to upper post!" , _func_);
		}
		return;
	}

    //normal pkg
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
	    case cs.CS_PROTO_LOGOUT_REQ:
			uid , exist := pconfig.Ckey2Uid[pclient.ClientKey];
			if !exist {
				log.Err("%s proto:%d but not login! key:%v" , _func_ , proto_id , pclient.ClientKey);
				return;
			}
	    	_ , ok := gmsg.SubMsg.(*cs.CSLogoutReq);
	    	if ok {
				log.Debug("%s recv proto:%d success! uid:%v" , _func_ , proto_id , uid);
				SendLogoutReq(pconfig , uid , ss.USER_LOGOUT_REASON_LOGOUT_CLIENT_EXIT);
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
