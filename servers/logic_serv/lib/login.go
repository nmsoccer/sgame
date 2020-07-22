package lib

import (
	"sgame/proto/ss"
	"sgame/servers/comm"
	"time"
)

type UserOnLine struct {
	login_ts int64
	hearbeat int64
	user_info *ss.UserInfo
}

type OnLineList struct {
	curr_online int
	user_map    map[int64]*UserOnLine
}

func RecvLoginReq(pconfig *Config, preq *ss.MsgLoginReq, msg []byte, from int) {
	var _func_ = "<RecvLoginReq>"
	log := pconfig.Comm.Log

	//log
	log.Debug("%s login: user:%s pass:%s device:%s c_key:%v from:%d", _func_, preq.GetName(), preq.GetPass(), preq.GetDevice(),
		preq.GetCKey(), from)

	//direct send
	ok := SendToDb(pconfig, msg)
	if !ok {
		log.Err("%s send failed!", _func_)
		return
	}
	//log.Debug("%s send to db success!", _func_)
	return
}

func RecvLoginRsp(pconfig *Config, prsp *ss.MsgLoginRsp, msg []byte) {
	var _func_ = "<RecvLoginRsp>"
	log := pconfig.Comm.Log

	//log
	log.Debug("%s result:%d c_key:%v user:%s", _func_, prsp.Result, prsp.CKey, prsp.Name)
    curr_ts := time.Now().Unix();
	/**Success Cache User*/
	switch prsp.Result {
	case ss.USER_LOGIN_RET_LOGIN_SUCCESS:
		puser_info := prsp.GetUserInfo()
		uid := puser_info.BasicInfo.Uid
		log.Info("%s login success! user:%s uid:%d last_login:%s last_logout:%s", _func_, prsp.Name, uid ,
			time.Unix(puser_info.BlobInfo.LastLoginTs,0).Format(comm.TIME_FORMAT_SEC) , time.Unix(puser_info.BlobInfo.LastLogoutTs , 0).Format(comm.TIME_FORMAT_SEC));
		//check exist
		if _, ok := pconfig.Users.user_map[uid]; ok {
			log.Info("%s user is already online! uid:%v", _func_, uid)
			break
		}

		//useronline
		ponline := new(UserOnLine)
		pconfig.Users.user_map[uid] = ponline
		ponline.user_info = puser_info
		ponline.login_ts = curr_ts;
		ponline.hearbeat = curr_ts;

		//init user_info
		if puser_info.BlobInfo == nil {
			puser_info.BlobInfo = new(ss.UserBlob);
		}
		InitUserInfo(pconfig , puser_info , uid);


		//update user_Info
		puser_info.BlobInfo.LastLoginTs = curr_ts;





		//add
		pconfig.Users.curr_online++

	default:
		//nothing to do
	}

	/**Back to Client*/
	ok := SendToConnect(pconfig, msg)
	if !ok {
		log.Err("%s send to connect failed! user:%s", _func_, prsp.Name)
		return
	}
	//log.Debug("%s send to connect success! user:%s", _func_, prsp.Name)
}

func RecvLogoutReq(pconfig *Config, plogout *ss.MsgLogoutReq) {
	var _func_ = "<RecvLogoutReq>"
	log := pconfig.Comm.Log
	log.Info("%s uid:%v reason:%v" , _func_ , plogout.Uid , plogout.Reason);

	//dispatch
	switch plogout.Reason {
	case ss.USER_LOGOUT_REASON_LOGOUT_CONN_CLOSED:
		//nothing to do
	case ss.USER_LOGOUT_REASON_LOGOUT_CLIENT_EXIT:
		//back to client
		SendLogoutRsp(pconfig, plogout.Uid, plogout.Reason, "fuck")
	default:
		//nothing to do
	}

    UserLogout(pconfig , plogout.Uid , plogout.Reason);
	return
}


func UserLogout(pconfig *Config , uid int64 , reason ss.USER_LOGOUT_REASON) {
	var _func_ = "<UserLogout>";
	log := pconfig.Comm.Log;

	//get online info
	ponline, ok := pconfig.Users.user_map[uid]
	if !ok {
		log.Info("%s uid:%d is offline!", _func_, uid)
		return
	}

	curr_ts := time.Now().Unix();
	//update user_info
	puser_info := ponline.user_info;
	puser_info.BlobInfo.LastLogoutTs = curr_ts;


	//save info
	var ss_msg ss.SSMsg
	ss_msg.ProtoType = ss.SS_PROTO_TYPE_LOGOUT_REQ
	body := new(ss.SSMsg_LogoutReq)
	body.LogoutReq = new(ss.MsgLogoutReq)
	body.LogoutReq.Uid = uid
	body.LogoutReq.Reason = reason
	body.LogoutReq.UserInfo = ponline.user_info
	ss_msg.MsgBody = body

	//pack and send
	buff, err := ss.Pack(&ss_msg)
	if err != nil {
		log.Err("%s pack failed! uid:%d reason:%d err:%v", _func_, uid, reason, err)
	} else {
		if !SendToDb(pconfig, buff) {
			log.Err("%s send to db failed! uid:%v reason:%v", _func_, uid, reason)
		}
	}

	//clear online
	pconfig.Users.curr_online -= 1
	if pconfig.Users.curr_online < 0 {
		pconfig.Users.curr_online = 0;
	}
	delete(pconfig.Users.user_map, uid)
	log.Info("%s done! uid:%v reason:%v curr_count:%d", _func_, uid, reason, pconfig.Users.curr_online)
	return;
}



//send logout rsp to connect
func SendLogoutRsp(pconfig *Config, uid int64, reason ss.USER_LOGOUT_REASON, msg string) {
	var _func_ = "<SendLogoutRsp>"
	log := pconfig.Comm.Log

	//msg
	var ss_msg ss.SSMsg
	ss_msg.ProtoType = ss.SS_PROTO_TYPE_LOGOUT_RSP
	body := new(ss.SSMsg_LogoutRsp)
	body.LogoutRsp = new(ss.MsgLogoutRsp)
	body.LogoutRsp.Uid = uid
	body.LogoutRsp.Reason = reason
	body.LogoutRsp.Msg = msg
	ss_msg.MsgBody = body

	//encode
	buff, err := ss.Pack(&ss_msg)
	if err != nil {
		log.Err("%s pack rsp failed! err:%v uid:%v reason:%v", _func_, err, uid, reason)
		return
	}

	//to connect
	ok := SendToConnect(pconfig, buff)
	if !ok {
		log.Err("%s send back failed! uid:%v reason:%v", _func_, uid, reason)
		return
	}
	log.Debug("%s send back success! uid:%v reason:%v", _func_, uid, reason)
	return
}

//check client timeout without heartbeat
func CheckClientTimeout(arg interface{}) {
	var _func_ = "<CheckClientTimeout>";
	pconfig , ok := arg.(*Config);
	if !ok {
		return;
	}
	log := pconfig.Comm.Log;

    //iter hearbeat
    curr_ts := time.Now().Unix();
    timeout_int := int64(pconfig.FileConfig.ClientTimeout);
    for uid , pinfo := range(pconfig.Users.user_map) {
        if pinfo.hearbeat + timeout_int < curr_ts {
        	log.Info("%s timeout uid:%d lastheart:%d" , _func_ , uid , pinfo.hearbeat);
        	UserLogout(pconfig , uid , ss.USER_LOGOUT_REASON_LOGOUT_CLIENT_TIMEOUT);
        	//send to client
        	SendLogoutRsp(pconfig , uid , ss.USER_LOGOUT_REASON_LOGOUT_CLIENT_TIMEOUT , "timeout");
		}
	}

    return;
}

func InitUserInfo(pconfig *Config , pinfo *ss.UserInfo , uid int64) {
	var _func_ = "<InitUserInfo>";
	log := pconfig.Comm.Log;

	log.Debug("%s uid:%d" , _func_ , uid);
	//init depot
	if pinfo.BlobInfo.Depot == nil {
		pinfo.BlobInfo.Depot = new(ss.UserDepot);
		InitUserDepot(pconfig, pinfo.BlobInfo.Depot , uid);
	}

}