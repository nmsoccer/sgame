package lib

import (
    "sgame/proto/ss"
)

type UserOnLine struct{
	user_info *ss.UserInfo;
}


type OnLineList struct {
	max_online int
	curr_online int
	user_map map[int64] *UserOnLine
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
	switch prsp.Result {
	case ss.USER_LOGIN_RET_LOGIN_SUCCESS:
		puser_info := prsp.GetUserInfo();
		uid := puser_info.BasicInfo.Uid;
		log.Info("%s login success! user:%s uid:%d" , _func_ , prsp.Name , uid);
		//check exist
		if _ , ok := pconfig.Users.user_map[uid]; ok {
			log.Info("%s user is already online! uid:%v" , _func_ , uid);
			break;
		}


        //useronline
		ponline := new(UserOnLine);
		pconfig.Users.user_map[uid] = ponline;
		ponline.user_info = puser_info;
        //add
        pconfig.Users.curr_online++;

	default:
		//nothing to do
	}
	
	
	/**Back to Client*/
	ok := SendToConnect(pconfig, msg);
	if !ok {
		log.Err("%s send to connect failed! user:%s" , _func_ , prsp.Name);
		return;
	}
	log.Debug("%s send to connect success! user:%s" , _func_ , prsp.Name);	
}

func RecvLogoutReq(pconfig *Config , plogout *ss.MsgLogoutReq , msg []byte) {
	var _func_ = "<RecvLogoutReq>";
	log := pconfig.Comm.Log;
	//log.Info("%s uid:%v reason:%v" , _func_ , plogout.Uid , plogout.Reason);

	//dispatch
	switch plogout.Reason {
	case ss.USER_LOGOUT_REASON_LOGOUT_CONN_CLOSED:
		//nothing to do
	case ss.USER_LOGOUT_REASON_LOGOUT_CLIENT_EXIT:
		//back to client
		SendLogoutRsp(pconfig , plogout.Uid , plogout.Reason , "fuck");
	default:
		//nothing to do
	}


	//clear online
	pconfig.Users.curr_online -= 1;
	delete(pconfig.Users.user_map , plogout.Uid);
	log.Info("%s success! uid:%v reason:%v curr_count:%d" , _func_ , plogout.Uid , plogout.Reason , pconfig.Users.curr_online);

	//send to db
	if !SendToDb(pconfig , msg) {
		log.Err("%s send to db failed! uid:%v reason:%v" , _func_ , plogout.Uid , plogout.Reason);
	}
	return;
}

//send logout rsp to connect
func SendLogoutRsp(pconfig *Config , uid int64 , reason ss.USER_LOGOUT_REASON , msg string) {
    var _func_ = "<SendLogoutRsp>";
    log := pconfig.Comm.Log;

    //msg
    var ss_msg ss.SSMsg;
    ss_msg.ProtoType = ss.SS_PROTO_TYPE_LOGOUT_RSP;
    body := new(ss.SSMsg_LogoutRsp);
    body.LogoutRsp = new(ss.MsgLogoutRsp);
    body.LogoutRsp.Uid = uid;
    body.LogoutRsp.Reason = reason;
    body.LogoutRsp.Msg = msg;
    ss_msg.MsgBody = body;

	//encode
	buff , err := ss.Pack(&ss_msg);
	if err != nil {
		log.Err("%s pack rsp failed! err:%v uid:%v reason:%v" , _func_ , err , uid , reason);
		return;
	}

	//to connect
	ok := SendToConnect(pconfig, buff);
	if !ok {
		log.Err("%s send back failed! uid:%v reason:%v" , _func_ , uid ,reason);
		return;
	}
	log.Debug("%s send back success! uid:%v reason:%v" , _func_ , uid , reason);
	return;
}