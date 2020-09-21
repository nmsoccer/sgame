package lib

import (
	"fmt"
	"sgame/proto/ss"
	"sgame/servers/comm"
	"strconv"
)

//user login using asynchronous
func RecvUserLoginReqAsync(pconfig *Config, preq *ss.MsgLoginReq, from int) {
	var _func_ = "<RecvUserLoginReq>"
	log := pconfig.Comm.Log

	log.Debug("%s user:%s pass:%s c_key:%d", _func_, preq.GetName(), preq.GetPass(), preq.GetCKey())
	//query pass
	cmd_arg := fmt.Sprintf(FORMAT_TAB_USER_GLOBAL, preq.Name)
	pconfig.RedisClient.RedisExeCmd(pconfig.Comm, cb_user_login_check_pass, []interface{}{preq, from},
		"HGETALL", cmd_arg)
}

//user login
func RecvUserLoginReq(pconfig *Config, preq *ss.MsgLoginReq, from int) {
	var _func_ = "<RecvUserLoginReq>"
	log := pconfig.Comm.Log

	log.Debug("%s user:%s pass:%s c_key:%d", _func_, preq.GetName(), preq.GetPass(), preq.GetCKey())
	//Sync Mod Must be In a routine
	go func(){
		//Get SyncHead
		phead := pconfig.RedisClient.AllocSyncCmdHead()
		if phead == nil {
			log.Err("%s alloc synchead faileed! uid:%d" , _func_ , preq.Uid)
			return
		}
		defer pconfig.RedisClient.FreeSyncCmdHead(phead)

		//check pass
		result , err := pconfig.RedisClient.RedisExeCmdSync(phead , "HGETALL", fmt.Sprintf(FORMAT_TAB_USER_GLOBAL, preq.Name))
		if err != nil {
			log.Err("%s query pass failed! name:%s" , _func_ , preq.Name)
			return
		}
		ok := user_login_check_pass(pconfig , result , preq , from)
		if !ok {
			return
		}

		//lock temp
		log.Debug("%s try to lock login. uid:%d name:%s" , _func_ , preq.Uid , preq.Name)
		tab_name := fmt.Sprintf(FORMAT_TAB_USER_LOGIN_LOCK_PREFIX+"%d" , preq.Uid)
		result , err = pconfig.RedisClient.RedisExeCmdSync(phead , "SET" , tab_name , preq.Uid , "EX" ,
			LOGIN_LOCK_LIFE, "NX")
		if err != nil {
			log.Err("%s lock login failed! name:%s uid:%d" , _func_ , preq.Name , preq.Uid)
			return
		}
		ok = user_login_lock(pconfig , result , preq , from)
		if !ok {
			return
		}

		//get user info
		log.Debug("%s ok! try to get user_info. uid:%d name:%s" , _func_ , preq.Uid , preq.Name)
		tab_name = fmt.Sprintf(FORMAT_TAB_USER_INFO_REFIX+"%d", preq.Uid)
		result , err = pconfig.RedisClient.RedisExeCmdSync(phead, "HGETALL", tab_name)
		if err != nil {
			log.Err("%s get user info failed! name:%s uid:%d" , _func_ , preq.Name , preq.Uid)
			return
		}
		pss_msg , ok := user_login_get_info(pconfig , result , preq , from)
		if !ok || pss_msg==nil {
			return
		}

		//update online_logic
		log.Debug("%s update online logic! uid:%d", _func_ , preq.Uid)
		tab_name = fmt.Sprintf(FORMAT_TAB_USER_INFO_REFIX+"%d", preq.Uid)
		result , err = pconfig.RedisClient.RedisExeCmdSync(phead , "HSET", tab_name, FIELD_USER_INFO_ONLINE_LOGIC , from)
		if err != nil {
			log.Err("%s update user online_logic failed! name:%s uid:%d" , _func_ , preq.Name , preq.Uid)
			return
		}
		user_login_update_online(pconfig , result , preq , from , pss_msg)
		log.Info("%s finish! uid:%d name:%s" , _func_ , preq.Uid , preq.Name)
	}()

}


//user logout
func RecvUserLogoutReq(pconfig *Config , preq *ss.MsgLogoutReq , from int) {
	var _func_ = "<RecvUserLogoutReq>"
	log := pconfig.Comm.Log

	//check info
	if preq.UserInfo == nil || preq.UserInfo.BasicInfo.Uid != preq.Uid {
		log.Err("%s fail! user_info not illegal! uid:%d reason:%d", _func_, preq.Uid, preq.Reason)
		return
	}
	log.Debug("%s user:%s uid:%d reason:%d", _func_, preq.UserInfo.BasicInfo.Name, preq.Uid, preq.Reason)

	//synchronise
	go func() {
		//Get SyncHead
		phead := pconfig.RedisClient.AllocSyncCmdHead()
		if phead == nil {
			log.Err("%s alloc synchead faileed! uid:%d" , _func_ , preq.Uid)
			return
		}
		defer pconfig.RedisClient.FreeSyncCmdHead(phead)

		//Exe Cmds
		user_tab := fmt.Sprintf(FORMAT_TAB_USER_INFO_REFIX+"%d", preq.Uid)
		puser_info := preq.UserInfo
		user_blob, err := ss.Pack(preq.UserInfo.BlobInfo)
		if err != nil {
			log.Err("%s save user_info failed! pack blob info fail! err:%v uid:%d", _func_, err, preq.Uid)
			return
		}
		result , err := pconfig.RedisClient.RedisExeCmdSync(phead , "HMSET", user_tab, "addr",
			puser_info.BasicInfo.Addr, "level", puser_info.BasicInfo.Level, FIELD_USER_INFO_ONLINE_LOGIC, -1 , "blob_info", string(user_blob))

		//Get Result
		if err != nil{
			log.Err("%s failed! err:%v uid:%d reason:%d", _func_, err , preq.Uid , preq.Reason)
			return
		}

		log.Info("%s done! ret:%v uid:%d reason:%d", _func_, result, preq.Uid , preq.Reason)
		return
	}()

}



/*---------------------------------STATIC FUNC-----------------------------*/
/*------------------------synchronise login func----------------------------*/
//@return next_step
func user_login_check_pass(pconfig *Config, result interface{}, preq *ss.MsgLoginReq , from_serv int) bool{
	var _func_ = "<user_login_check_pass>"
	log := pconfig.Comm.Log

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
			return false
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
		}
		return true
	}

	/*Back to Client*/
	//fill
	err := comm.FillSSPkg(&ss_msg , ss.SS_PROTO_TYPE_LOGIN_RSP , prsp)
	if err != nil {
		log.Err("%s gen ss failed! err:%v", _func_, err)
		return false
	}

	//send
	SendToServ(pconfig, from_serv , &ss_msg)
	return false
}

//lock login stat
//@return next_step
func user_login_lock(pconfig *Config, result interface{}, preq *ss.MsgLoginReq , from_serv int) bool {
	var _func_ = "<user_login_lock>"
	log := pconfig.Comm.Log

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
		return false
	}

	return true
}

//get user detail inf
//@return pss_msg(if success) , next_step
func user_login_get_info(pconfig *Config, result interface{}, preq *ss.MsgLoginReq , from_serv int) (*ss.SSMsg , bool) {
	var _func_ = "<user_login_get_info>"
	log := pconfig.Comm.Log


	/*create rsp */
	pss_msg := new(ss.SSMsg)
	prsp := new(ss.MsgLoginRsp)
	prsp.CKey = preq.CKey
	prsp.Name = preq.Name
	prsp.Uid = preq.Uid

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

		if uid != preq.Uid {
			log.Err("%s fail! uid not match! %d<->%d" , _func_ , uid , preq.Uid)
			prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR
			break
		}
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
			return nil , false
		}

		return pss_msg , true
	}

	/* Err will Back to Client*/
	//fill
	err := comm.FillSSPkg(pss_msg , ss.SS_PROTO_TYPE_LOGIN_RSP , prsp)
	if err != nil {
		log.Err("%s gen ss failed! err:%v", _func_, err)
		return nil , false
	}

	//send
	SendToServ(pconfig, from_serv , pss_msg)
	return nil , false
}


//cb_arg={0:preq 1:from_server 2:pss_msg}
func user_login_update_online(pconfig *Config, result interface{}, preq *ss.MsgLoginReq , from_serv int , pss_msg *ss.SSMsg) {
	var _func_ = "<user_login_update_online>"
	log := pconfig.Comm.Log
	uid := preq.Uid

	/*---------result handle--------------*/
	/*Get Result*/
	ret_code, err := comm.Conv2Int(result)
	if err != nil {
		log.Err("%s conv result failed! uid:%d err:%v", _func_, uid, err)
		return
	}
	log.Debug("%s ret_code:%d name:%s uid:%d", _func_, ret_code, preq.Name , uid)

	//Back to Client
	SendToServ(pconfig, from_serv , pss_msg)
}

/*------------------------asyn login func deprecated----------------------------*/
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