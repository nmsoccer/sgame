package lib

import (
	"fmt"
	"sgame/proto/ss"
	"sgame/servers/comm"
	"strconv"
)

//user login
func RecvUserLoginReq(pconfig *Config, preq *ss.MsgLoginReq, from int) {
	var _func_ = "<RecvUserLoginReq>"
	log := pconfig.Comm.Log

	log.Debug("%s user:%s pass:%s c_key:%d", _func_, preq.GetName(), preq.GetPass(), preq.GetCKey())
	//query pass
	cmd_arg := fmt.Sprintf(FORMAT_TAB_USER_GLOBAL, preq.Name)
	pconfig.RedisClient.RedisExeCmd(pconfig.Comm, cb_user_login_check_pass, []interface{}{preq, from},
		"HGETALL", cmd_arg)
}

//user logout
func RecvUserLogoutReq(pconfig *Config, preq *ss.MsgLogoutReq, from int) {
	var _func_ = "<RecvUserLogoutReq>"
	log := pconfig.Comm.Log

	//check info
	if preq.UserInfo == nil || preq.UserInfo.BasicInfo.Uid != preq.Uid {
		log.Err("%s fail! user_info not illegal! uid:%d reason:%d", _func_, preq.Uid, preq.Reason)
		return
	}
	log.Debug("%s user:%s uid:%d reason:%d", _func_, preq.UserInfo.BasicInfo.Name, preq.Uid, preq.Reason)

	cb_arg := []interface{}{from, preq.Uid, preq.Reason}

	//save user info
	user_tab := fmt.Sprintf(FORMAT_TAB_USER_INFO_REFIX+"%d", preq.Uid)
	puser_info := preq.UserInfo
	user_blob, err := ss.Pack(preq.UserInfo.BlobInfo)
	if err != nil {
		log.Err("%s save user_info failed! pack blob info fail! err:%v uid:%d", _func_, err, preq.Uid)
		return
	}
	pconfig.RedisClient.RedisExeCmd(pconfig.Comm, cb_save_user_logout, cb_arg, "HMSET", user_tab, "addr",
		puser_info.BasicInfo.Addr, "level", puser_info.BasicInfo.Level, "online_logic", -1 , "blob_info", string(user_blob))
	return
}

/*---------------------------------STATIC FUNC-----------------------------*/
//cb_arg={0:preq 1:from_server}
func cb_user_login_check_pass(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_user_login_check_pass>"
	log := comm_config.Log
	pconfig, ok := comm_config.ServerCfg.(*Config)
	if !ok {
		log.Err("%s get config failed! cb:%v", _func_, cb_arg)
		return
	}

	//conv cb-arg
	preq, ok := cb_arg[0].(*ss.MsgLoginReq)
	if !ok {
		log.Err("%s conv req failed! cb:%v", _func_, cb_arg)
		return
	}

	from_serv, ok := cb_arg[1].(int)
	if !ok {
		log.Err("%s conv from failed! cb:%v", _func_, cb_arg)
		return
	}

	//log.Debug("%s user:%s pass:%s c_key:%v", _func_, preq.GetName(), preq.GetPass(), preq.GetCKey())
	//check error
	if err, ok := result.(error); ok {
		log.Err("%s reply error! name:%s err:%v", _func_, preq.Name, err)
		return
	}

	//rsp
	var ss_msg ss.SSMsg
	prsp := new(ss.MsgLoginRsp)
	prsp.CKey = preq.CKey
	prsp.Name = preq.Name


	//do while 0
	for {
		//check result may need reg
		if result == nil {
			log.Info("%s no user:%s exist!", _func_, preq.Name)
			prsp.Result = ss.USER_LOGIN_RET_LOGIN_EMPTY
			break
		}

		//conv result
		sm, err := comm.Conv2StringMap(result)
		if err != nil {
			log.Err("%s conv result failed! err:%v", _func_, err)
			return
		}

		pass := sm["pass"]
		uid := sm["uid"]
		salt := sm["salt"]
		log.Debug("%s try to check pass! uid:%s", _func_, uid)
		//check pass
		enc_pass := comm.EncPassString(preq.Pass , salt)
		if enc_pass != pass {
			log.Info("%s pass not matched! user:%s c_key:%v ", _func_, preq.Name, preq.CKey)
			prsp.Result = ss.USER_LOGIN_RET_LOGIN_PASS
			break
		}

		//sucess. get user info
		if preq.Uid == 0 { //default role
			//conv uid
			preq.Uid , err = strconv.ParseInt(uid , 10 , 64)
			if err != nil {
				log.Err("%s conv uid failed! err:%v uid:%s" , _func_ , err , uid)
				prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR
				break;
			}

			//try to lock
			log.Debug("%s try to lock login. uid:%d name:%s" , _func_ , preq.Uid , preq.Name)
			tab_name := fmt.Sprintf(FORMAT_TAB_USER_LOGIN_LOCK_PREFIX+"%s" , uid)
			pconfig.RedisClient.RedisExeCmd(pconfig.Comm , cb_user_login_lock , cb_arg , "SET" , tab_name , uid , "EX" ,
				LOGIN_LOCK_LIFE, "NX")
			//tab_name := fmt.Sprintf(FORMAT_TAB_USER_INFO_REFIX+"%s", uid)
			//pconfig.RedisClient.RedisExeCmd(pconfig.Comm, cb_user_login_get_info, cb_arg, "HGETALL", tab_name)
		}
		return
	}

	/*Back to Client*/
	//fill
	err := comm.FillSSPkg(&ss_msg , ss.SS_PROTO_TYPE_LOGIN_RSP , prsp)
	if err != nil {
		log.Err("%s gen ss failed! err:%v", _func_, err)
		return
	}

	//send
	ok = SendToServ(pconfig, from_serv , &ss_msg)
	if !ok {
		log.Err("%s send back to %d failed!", _func_, from_serv)
		return
	}
	log.Debug("%s send back to %d success!", _func_, from_serv)
	return
}


//cb_arg={0:preq 1:from_server}
func cb_user_login_lock(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_user_login_lock>"
	log := comm_config.Log

	/*---------mostly common logic--------------*/
	pconfig, ok := comm_config.ServerCfg.(*Config)
	if !ok {
		log.Err("%s get config failed! cb:%v", _func_, cb_arg)
		return
	}

	/*convert callback arg*/
	preq, ok := cb_arg[0].(*ss.MsgLoginReq)
	if !ok {
		log.Err("%s conv cb failed!", _func_)
		return
	}

	from_serv, ok := cb_arg[1].(int)
	if !ok {
		log.Err("%s conv from failed! cb:%v", _func_, cb_arg)
		return
	}

	log.Debug("%s user:%s c_key:%v", _func_, preq.GetName(), preq.GetCKey())
	/*---------result handle--------------*/
	//check error
	if err, ok := result.(error); ok {
		log.Err("%s reply error! name:%s err:%v", _func_, preq.Name, err)
		return
	}

	//check result
	if result == nil { //in login process
		log.Err("%s is in login process! uid:%d name:%s" , _func_ , preq.Uid , preq.Name)
		//rsp
		var ss_msg ss.SSMsg
		prsp := new(ss.MsgLoginRsp)
		prsp.CKey = preq.CKey
		prsp.Name = preq.Name
		prsp.Result = ss.USER_LOGIN_RET_LOGIN_MULTI_ON

		if err := comm.FillSSPkg(&ss_msg , ss.SS_PROTO_TYPE_LOGIN_RSP , prsp); err != nil {
			log.Err("%s gen ss failed! err:%v name:%s" , _func_ , err , preq.Name)
		} else {
            SendToServ(pconfig , from_serv , &ss_msg)
		}
		return
	}


	//get user info
	log.Debug("%s ok! try to get user_info. uid:%d name:%s" , _func_ , preq.Uid , preq.Name)
	tab_name := fmt.Sprintf(FORMAT_TAB_USER_INFO_REFIX+"%d", preq.Uid)
	pconfig.RedisClient.RedisExeCmd(pconfig.Comm, cb_user_login_get_info, cb_arg, "HGETALL", tab_name)
}

//cb_arg={0:preq 1:from_server}
func cb_user_login_get_info(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_user_login_get_info>"
	log := comm_config.Log

	/*---------mostly common logic--------------*/
	pconfig, ok := comm_config.ServerCfg.(*Config)
	if !ok {
		log.Err("%s get config failed! cb:%v", _func_, cb_arg)
		return
	}

	/*convert callback arg*/
	preq, ok := cb_arg[0].(*ss.MsgLoginReq)
	if !ok {
		log.Err("%s conv cb failed!", _func_)
		return
	}

	from_serv, ok := cb_arg[1].(int)
	if !ok {
		log.Err("%s conv from failed! cb:%v", _func_, cb_arg)
		return
	}

	log.Debug("%s user:%s c_key:%v", _func_, preq.GetName(), preq.GetCKey())
	/*---------result handle--------------*/
	//check error
	if err, ok := result.(error); ok {
		log.Err("%s reply error! name:%s err:%v", _func_, preq.Name, err)
		return
	}

	/*create rsp */
	pss_msg := new(ss.SSMsg)
	prsp := new(ss.MsgLoginRsp)
	prsp.CKey = preq.CKey
	prsp.Name = preq.Name

	//do while 0
	for {
		//check result
		if result == nil {
			log.Err("%s no user detail:%s exist!", _func_, preq.Name)
			prsp.Result = ss.USER_LOGIN_RET_LOGIN_EMPTY
			break
		}

		//conv result
		sm, err := comm.Conv2StringMap(result)
		if err != nil {
			log.Err("%s conv result failed! err:%v", _func_, err)
			prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR
			break
		}

		//Get User Info
		prsp.UserInfo = new(ss.UserInfo)
		prsp.UserInfo.BasicInfo = new(ss.UserBasic)
		pbasic := prsp.UserInfo.BasicInfo
		puser_blob := new(ss.UserBlob)

		var uid int64
        var online_logic  = -1

		//uid
		if v, ok := sm["uid"]; ok {
			uid, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				log.Err("%s conv uid failed! err:%v uid:%s", _func_, err, v)
				prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR
				break
			}
		} else {
			log.Err("%s no uid found of user:%s", _func_, preq.Name)
			prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR
			break
		}
		prsp.Uid = uid
		pbasic.Uid = uid


		//online_logic
		if v , ok := sm["online_logic"]; ok {
			online_logic , err = strconv.Atoi(v);
			if err != nil {
				log.Err("%s conv online-logic failed! err:%v" , _func_ , err);
				prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR
			}
		} else {
			log.Err("%s conv online-logic not exist! uid:%d" , _func_ , uid);
			prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR
		}
		//check online
		//already login at other logic should kick first
		if online_logic >= 0 && online_logic != from_serv {
			log.Info("%s user:%s login at other logic server %d now logic:%d kick first!" , _func_ , preq.Name , online_logic ,
				from_serv);
			prsp.Result = ss.USER_LOGIN_RET_LOGIN_MULTI_ON;
			prsp.OnlineLogic = int32(online_logic);
			break;
		}


		//role_name
		if v, ok := sm["name"]; ok {
			pbasic.Name = v
		} else {
			log.Err("%s no role_name found of uid:%d", _func_, uid)
			prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR
			break
		}

		//age
		if v, ok := sm["age"]; ok {
			age, err := strconv.Atoi(v)
			if err != nil {
				log.Err("%s conv age failed! err:%v age:%s", _func_, err, v)
				prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR
				break
			}
			pbasic.Age = int32(age)
		}

		//sex
		if v, ok := sm["sex"]; ok {
			sex, err := strconv.Atoi(v)
			if err != nil {
				log.Err("%s conv sex failed! err:%v sex:%s", _func_, err, v)
				prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR
				break
			}
			if sex == 1 {
				pbasic.Sex = true //male; false:female
			} else {
				pbasic.Sex = false
			}
		}

		//addr
		if v, ok := sm["addr"]; ok {
			pbasic.Addr = v
		}

		//level
		if v, ok := sm["level"]; ok {
			level, err := strconv.Atoi(v)
			if err != nil {
				log.Err("%s conv level failed! err:%v level:%s", _func_, err, v)
				prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR
				break
			}
			pbasic.Level = int32(level)
		}

		//blob info
		if v, ok := sm["blob_info"]; ok {
			err = ss.UnPack([]byte(v), puser_blob)
			if err != nil {
				log.Err("%s unpack user_blob failed! err:%v uid:%d", _func_, err, uid)
				prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR
				break
			}
		} else { //set blob_info default value
			puser_blob.Exp = 100
		}
		prsp.UserInfo.BlobInfo = puser_blob

		//Fullfill
		prsp.Result = ss.USER_LOGIN_RET_LOGIN_SUCCESS
		log.Debug("%s success! user:%s uid:%v", _func_, pbasic.Name, pbasic.Uid)

		//msg
		err = comm.FillSSPkg(pss_msg , ss.SS_PROTO_TYPE_LOGIN_RSP , prsp)
		if err != nil {
			log.Err("%s gen ss failed! err:%v" , _func_ , err)
			return
		}

		//update online_logic
		log.Debug("%s update online logic!", _func_)
		tab_name := fmt.Sprintf(FORMAT_TAB_USER_INFO_REFIX+"%d", uid)
		pconfig.RedisClient.RedisExeCmd(pconfig.Comm, cb_update_online, append(cb_arg, pss_msg), "HSET", tab_name, "online_logic", from_serv)
		return
	}

	/* Err will Back to Client*/
	//fill
	err := comm.FillSSPkg(pss_msg , ss.SS_PROTO_TYPE_LOGIN_RSP , prsp)
	if err != nil {
		log.Err("%s gen ss failed! err:%v", _func_, err)
		return
	}

	//send
	ok = SendToServ(pconfig, from_serv , pss_msg)
	if !ok {
		log.Err("%s send back to %d failed!", _func_, from_serv)
	}
}


//cb_arg={0:preq 1:from_server 2:pss_msg}
func cb_update_online(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_update_online>"
	log := comm_config.Log

	/*---------mostly common logic--------------*/
	pconfig, ok := comm_config.ServerCfg.(*Config)
	if !ok {
		log.Err("%s get config failed! cb:%v", _func_, cb_arg)
		return
	}

	/*convert callback arg*/
	preq, ok := cb_arg[0].(*ss.MsgLoginReq)
	if !ok {
		log.Err("%s conv req failed! cb:%v", _func_, cb_arg)
		return
	}

	from_serv, ok := cb_arg[1].(int)
	if !ok {
		log.Err("%s conv from failed! cb:%v", _func_, cb_arg)
		return
	}

	pss_msg , ok := cb_arg[2].(*ss.SSMsg)
	if !ok {
		log.Err("%s conv ss_msg failed! cb:%v", _func_, cb_arg)
		return
	}

	uid := pss_msg.GetLoginRsp().Uid
	/*---------result handle--------------*/
	//check error
	if err, ok := result.(error); ok {
		log.Err("%s reply error! uid:%d err:%v", _func_, uid , err)
		return
	}

	/*Get Result*/
	ret_code, err := comm.Conv2Int(result)
	if err != nil {
		log.Err("%s conv result failed! uid:%d err:%v", _func_, uid, err)
		return
	}
	log.Debug("%s ret_code:%d name:%s uid:%d", _func_, ret_code, preq.Name , uid)

	//Back to Client
	ok = SendToServ(pconfig, from_serv , pss_msg)
	if !ok {
		log.Err("%s send back to %d failed!", _func_, from_serv)
	}
}


//
//cb_arg := []interface{}{from , Uid , Reason};
func cb_update_online_logout(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_update_online_logout>"
	log := comm_config.Log

	//Get Result
	if err, ok := result.(error); ok {
		log.Err("%s failed! err:%v uid:%v reason:%v", _func_, err, cb_arg[1], cb_arg[2])
		return
	}

	log.Info("%s done! ret:%v uid:%v reason:%v", _func_, result, cb_arg[1], cb_arg[2])
	return
}

//cb_arg := []interface{}{from , Uid , Reason};
func cb_save_user_logout(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_save_user_logout>"
	log := comm_config.Log

	//Get Result
	if err, ok := result.(error); ok {
		log.Err("%s failed! err:%v uid:%v reason:%v", _func_, err, cb_arg[1], cb_arg[2])
		return
	}

	log.Info("%s done! ret:%v uid:%v reason:%v", _func_, result, cb_arg[1], cb_arg[2])
	return
}
