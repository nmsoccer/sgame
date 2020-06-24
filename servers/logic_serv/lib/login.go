package lib

import (
    "sgame/proto/ss"
)

type UserData struct{
	user_info ss.UserInfo;
}


type UserOnLine struct {
	max_online int
	curr_online int
	users []*UserData
}



func RecvLoginReq(pconfig *Config , preq *ss.MsgLoginReq , msg []byte , from int) {
    var _func_ = "<RecvLoginReq>";
    log := pconfig.Comm.Log;
    
    //log
    log.Debug("%s login: user:%s pass:%s device:%s c_key:%v from:%d" , _func_ , preq.GetName() , preq.GetPass() , preq.GetDevice() , 
    	preq.GetCKey() , from);	

    //direct send
    ok := SendToDb(pconfig, msg);
    if !ok {
    	log.Err("%s send failed!" , _func_);
    	return;
    }
    log.Debug("%s send to db success!" , _func_);
    return;
}

func RecvLoginRsp(pconfig *Config , prsp *ss.MsgLoginRsp , msg []byte) {
	var _func_ = "<RecvLoginRsp>";
	log := pconfig.Comm.Log;
	
	//log
	log.Debug("%s result:%d c_key:%v user:%s" , _func_ , prsp.Result , prsp.CKey , prsp.Name);
	
	/**Success Cache User*/
	if prsp.Result == ss.USER_LOGIN_RET_LOGIN_SUCCESS {
		log.Info("%s login success! user:%s uid:%d" , _func_ , prsp.Name , prsp.GetUserInfo().GetBasicInfo().Uid);
	}
	
	
	/**Back to Client*/
	ok := SendToConnect(pconfig, msg);
	if !ok {
		log.Err("%s send to connect failed! user:%s" , _func_ , prsp.Name);
		return;
	}
	log.Debug("%s send to connect success! user:%s" , _func_ , prsp.Name);	
}