package lib

import (
	"fmt"
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

/*
Get UserOnLine Info by Uid
*/
func GetUserInfo(pconfig *Config, uid int64) *UserOnLine {
	pinfo, exist := pconfig.Users.user_map[uid]
	if !exist {
		return nil
	}
	return pinfo
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
	case ss.USER_LOGIN_RET_LOGIN_MULTI_ON: //kick other logic role firstly
		log.Info("%s login at other logic:%d. will kick it first! uid:%d" , _func_ , prsp.OnlineLogic , prsp.Uid);
        //SendTransLogicKick(pconfig , prsp);
        SendDupUserKick(pconfig , prsp)


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
		log_content := fmt.Sprintf("%d|%s|LoginFlow|%d|%s|%s|%d" , pconfig.ProcId , pconfig.ProcName , uid , prsp.Name ,
			prsp.UserInfo.BasicInfo.Addr,curr_ts)
		pconfig.NetLog.Log("|" , log_content)


		//repack ss_msg
		var ss_msg ss.SSMsg

	    //fill
	    err := comm.FillSSPkg(&ss_msg , ss.SS_PROTO_TYPE_LOGIN_RSP , prsp)
		if err != nil {
			log.Err("%s gen ss_msg failed! err:%v" , _func_ , err)
			return
		}

		SendToConnect(pconfig , &ss_msg)
		return
	default:
		//nothing to do
	}

	/**Back to Client*/
	SendToConnect(pconfig, msg)
}

func SendDupUserKick(pconfig *Config, prsp *ss.MsgLoginRsp) {
	var _func_ = "<SendDupUserKick>"
	log := pconfig.Comm.Log

	if prsp.Uid <= 0 {
		log.Err("%s failed! uid empty! name:%s", _func_, prsp.Name)
		return
	}

	//gen disp msg
	pkick := new(ss.MsgDispKickDupUser)
	pkick.TargetUid = prsp.Uid
	pss_msg, err := comm.GenDispMsg(ss.DISP_MSG_TARGET_NON_SERVER, ss.DISP_MSG_METHOD_SPEC, ss.DISP_PROTO_TYPE_KICK_DUPLICATE_USER, int(prsp.OnlineLogic),
		pconfig.ProcId, 0 , pkick)
	if err != nil {
		log.Err("%s generate disp msg failed! uid:%d err:%v", _func_, err)
		return
	}

	//pack
	enc_data, err := ss.Pack(pss_msg)
	if err != nil {
		log.Err("%s enc failed! err:%v uid:%d", _func_, err, prsp.Uid)
		return
	}

	//send
	if !SendToDisp(pconfig, prsp.Uid, enc_data) {
		log.Err("%s send to disp failed! uid:%d", _func_, prsp.Uid)
	}
	log.Debug("%s send to disp success! name:%s uid:%d", _func_, prsp.Name, prsp.Uid)
}

//recv kickout from other logic-serv
func RecvDupUserKick(pconfig *Config, pmsg *ss.MsgDispKickDupUser, from int) {
	var _func_ = "<RecvDupUserKick>"
	log := pconfig.Comm.Log

	//check arg
	if pmsg == nil {
		log.Err("%s req nil!", _func_)
		return
	}

	log.Info("%s will kickout uid:%d from:%d", _func_, pmsg.TargetUid, from)
	//logout
	var logout ss.MsgLogoutReq
	logout.Uid = pmsg.TargetUid
	logout.Reason = ss.USER_LOGOUT_REASON_LOGOUT_SERVER_KICK_RECONN
	RecvLogoutReq(pconfig, &logout)
}




func RecvLogoutReq(pconfig *Config, plogout *ss.MsgLogoutReq) {
	var _func_ = "<RecvLogoutReq>"
	log := pconfig.Comm.Log
	log.Info("%s uid:%v reason:%v" , _func_ , plogout.Uid , plogout.Reason);

	//dispatch
	switch plogout.Reason {
	case ss.USER_LOGOUT_REASON_LOGOUT_CONN_CLOSED:
		//return. wait for re-connect
		return;
	case ss.USER_LOGOUT_REASON_LOGOUT_CLIENT_EXIT:
		//back to client
		SendLogoutRsp(pconfig, plogout.Uid, plogout.Reason, "fuck")
	case ss.USER_LOGOUT_REASON_LOGOUT_SERVER_KICK_RECONN:
		//back to client
		SendLogoutRsp(pconfig , plogout.Uid , plogout.Reason , "login-other");
	default:
		//nothing to do
	}

	//save user info and clear login stat
    UserLogout(pconfig , plogout.Uid , plogout.Reason);
	return
}


func UserLogout(pconfig *Config , uid int64 , reason ss.USER_LOGOUT_REASON) {
	var _func_ = "<UserLogout>";
	log := pconfig.Comm.Log;

	//get online info
	ponline := GetUserInfo(pconfig, uid)
	if ponline == nil {
		log.Info("%s uid:%d is offline! will logout off-mode", _func_, uid)
		//will modify online-logic
		var ss_msg ss.SSMsg
		pLogoutReq := new(ss.MsgLogoutReq)
		pLogoutReq.Uid = uid
		pLogoutReq.Reason = ss.USER_LOGOUT_REASON_LOGOUT_OFFLINE_USER
		err := comm.FillSSPkg(&ss_msg , ss.SS_PROTO_TYPE_LOGOUT_REQ , pLogoutReq)
		if err != nil {
			log.Err("%s pack failed! uid:%d reason:%d err:%v", _func_, uid, reason, err)
		}
		SendToDb(pconfig, &ss_msg)
		return
	}

	curr_ts := time.Now().Unix();
	//update user_info
	puser_info := ponline.user_info;
	puser_info.BlobInfo.LastLogoutTs = curr_ts;


	//save info
	var ss_msg ss.SSMsg
	pLogoutReq := new(ss.MsgLogoutReq)
	pLogoutReq.Uid = uid
	pLogoutReq.Reason = reason
	pLogoutReq.UserInfo = ponline.user_info

	//fill and send
	err := comm.FillSSPkg(&ss_msg , ss.SS_PROTO_TYPE_LOGOUT_REQ , pLogoutReq)
	if err != nil {
		log.Err("%s pack failed! uid:%d reason:%d err:%v", _func_, uid, reason, err)
	} else {
		if !SendToDb(pconfig, &ss_msg) {
			log.Err("%s send to db failed! uid:%v reason:%v", _func_, uid, reason)
		}
	}

	//nlog
	log_content := fmt.Sprintf("%d|%s|LogoutFlow|%d|%s|%d|%d" , pconfig.ProcId , pconfig.ProcName , uid , puser_info.BasicInfo.Name ,
		reason , curr_ts)
	pconfig.NetLog.Log("|" , log_content)

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
	pLogoutRsp := new(ss.MsgLogoutRsp)
	pLogoutRsp.Uid = uid
	pLogoutRsp.Reason = reason
	pLogoutRsp.Msg = msg


	//fill
	err := comm.FillSSPkg(&ss_msg , ss.SS_PROTO_TYPE_LOGOUT_RSP , pLogoutRsp)
	if err != nil {
		log.Err("%s gen ss failed! err:%v uid:%v reason:%v", _func_, err, uid, reason)
		return
	}

	//to connect
	SendToConnect(pconfig, &ss_msg)
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