package lib

import (
	"sgame/proto/cs"
	"sgame/proto/ss"
)

func SendLoginReq(pconfig *Config , client_key int64 , plogin_req *cs.CSLoginReq) {
	var _func_ = "<SendLoginReq>";
	log := pconfig.Comm.Log;
	
	log.Debug("%s send login pkg to logic! user:%s device:%s" , _func_ , plogin_req.Name ,plogin_req.Device);
	//create pkg
	var ss_msg ss.SSMsg;
	ss_msg.ProtoType = ss.SS_PROTO_TYPE_LOGIN_REQ;
	body := new(ss.SSMsg_LoginReq);
	body.LoginReq = new(ss.MsgLoginReq);
	body.LoginReq.CKey = client_key;
	body.LoginReq.Name = plogin_req.Name;
	body.LoginReq.Pass = plogin_req.Pass;
	body.LoginReq.Device = plogin_req.Device;
	ss_msg.MsgBody = body;
	
	//pack
	coded , err := ss.Pack(&ss_msg);
	if err != nil {
		log.Err("%s pack failed! err:%v ckey:%v" , _func_ , err , client_key);
		return;
	}
	
	//send
	ok := SendToLogic(pconfig, coded);
	if !ok {
		log.Err("%s send failed! client_key:%v" , _func_ , client_key);
		return;
	}
    log.Debug("%s send success!" , _func_);
	return;
}


func RecvLoginRsp(pconfig *Config , prsp *ss.MsgLoginRsp) {
	var _func_ = "<RecvLoginRsp>";
	log := pconfig.Comm.Log;
	
	//log
	log.Debug("%s result:%d user:%s c_key:%v" , _func_ , prsp.Result , prsp.Name , prsp.CKey);
	
	//response
	var gmsg cs.GeneralMsg;
	gmsg.ProtoId = cs.CS_PROTO_LOGIN_RSP;
	psub := new(cs.CSLoginRsp);
	gmsg.SubMsg = psub;
	
	//gmsg
	psub.Result = int(prsp.Result);
	psub.Name = prsp.Name;
	  //success
	if prsp.Result == ss.USER_LOGIN_RET_LOGIN_SUCCESS {
		psub.Basic.Uid = prsp.GetUserInfo().BasicInfo.Uid;
		psub.Basic.Name = prsp.Name;
		psub.Basic.Addr = prsp.UserInfo.BasicInfo.Addr;
		if prsp.UserInfo.BasicInfo.Sex {
			psub.Basic.Sex = 1;
		} else {
			psub.Basic.Sex = 0;
		}
		
		//create map refer
		pconfig.Ckey2Uid[prsp.CKey] = psub.Basic.Uid;
		pconfig.Uid2Ckey[psub.Basic.Uid] = prsp.CKey;
	}
	
	//to client
	SendToClient(pconfig, prsp.CKey, &gmsg);		
}

func SendLogoutReq(pconfig *Config , uid int64 , reason ss.USER_LOGOUT_REASON) {
	var _func_ = "<SendLogoutReq>";
	log := pconfig.Comm.Log;
	var ss_msg ss.SSMsg;
	//construct
	ss_msg.ProtoType = ss.SS_PROTO_TYPE_LOGOUT_REQ;
	body := new(ss.SSMsg_LogoutReq);
	body.LogoutReq = new(ss.MsgLogoutReq);
    body.LogoutReq.Uid = uid;
    body.LogoutReq.Reason = reason;
    ss_msg.MsgBody = body;

	//pack
	coded , err := ss.Pack(&ss_msg);
	if err != nil {
		log.Err("%s pack failed! err:%v uid:%v reason:%v" , _func_ , err , uid , reason);
		return;
	}

	//send
	ok := SendToLogic(pconfig, coded);
	if !ok {
		log.Err("%s send failed! uid:%v reason:%v" , _func_ , uid , reason);
		return;
	}
	log.Debug("%s send success!" , _func_);
	return;
}

func RecvLogoutRsp(pconfig *Config , prsp *ss.MsgLogoutRsp) {
	var _func_ = "<RecvLogoutRsp>";
	log := pconfig.Comm.Log;
	log.Info("%s uid:%v reason:%v msg:%s" , _func_ , prsp.Uid , prsp.Reason , prsp.Msg);

	//response
	var gmsg cs.GeneralMsg;
	gmsg.ProtoId = cs.CS_PROTO_LOGOUT_RSP;
	psub := new(cs.CSLogoutRsp);
	gmsg.SubMsg = psub;

    //fill
    psub.Msg = prsp.Msg;
    psub.Result = 0;
    psub.Uid = prsp.Uid;

    //to client
    c_key , ok := pconfig.Uid2Ckey[prsp.Uid];
    if !ok {
    	log.Err("%s no c_key found! uid:%v reason:%v msg:%s" , _func_ , prsp.Uid , prsp.Reason , prsp.Msg);
    	return;
	}
	SendToClient(pconfig , c_key , &gmsg);

    //clear map
    delete(pconfig.Uid2Ckey , prsp.Uid);
    delete(pconfig.Ckey2Uid , c_key);
}