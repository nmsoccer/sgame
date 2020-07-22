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
	cmd_arg := fmt.Sprintf(FORMAT_TAB_USER_GLOBAL , preq.Name);
	pconfig.RedisClient.RedisExeCmd(pconfig.Comm, cb_user_login_check_pass, []interface{}{preq, from},
		"HGETALL", cmd_arg)
}

//user logout
func RecvUserLogoutReq(pconfig *Config , preq *ss.MsgLogoutReq , from int) {
	var _func_ = "<RecvUserLogoutReq>";
	log := pconfig.Comm.Log;

    //check info
    if preq.UserInfo == nil || preq.UserInfo.BasicInfo.Uid != preq.Uid {
    	log.Err("%s fail! user_info not illegal! uid:%d reason:%d" , _func_ , preq.Uid , preq.Reason);
    	return;
	}
	log.Debug("%s user:%s uid:%d reason:%d" , _func_ , preq.UserInfo.BasicInfo.Name , preq.Uid , preq.Reason);

	cb_arg := []interface{}{from , preq.Uid , preq.Reason};
	//update online
	global_tab := fmt.Sprintf(FORMAT_TAB_USER_GLOBAL , preq.UserInfo.BasicInfo.Name);
	pconfig.RedisClient.RedisExeCmd(pconfig.Comm , cb_update_online_logout , cb_arg , "HSET" , global_tab , "online_logic" , -1);


	//save user info
	user_tab := fmt.Sprintf(FORMAT_TAB_USER_INFO_REFIX+"%d" , preq.Uid);
	puser_info := preq.UserInfo;
	user_blob  , err := ss.Pack(preq.UserInfo.BlobInfo);
	if err != nil {
		log.Err("%s save user_info failed! pack blob info fail! err:%v uid:%d" , _func_ , err , preq.Uid);
		return;
	}
	pconfig.RedisClient.RedisExeCmd(pconfig.Comm , cb_save_user_logout , cb_arg , "HMSET" , user_tab , "addr" ,
		puser_info.BasicInfo.Addr , "level" , puser_info.BasicInfo.Level , "blob_info" , string(user_blob));
    return;
}

/*---------------------------------STATIC FUNC-----------------------------*/
//cb_arg={0:preq 1:from_server}
func cb_user_login_check_pass(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_user_login_check_pass>"
	var ss_msg ss.SSMsg
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
	if err , ok := result.(error); ok {
        log.Err("%s reply error! name:%s err:%v" , _func_ , preq.Name , err);
        return;
	}

	//rsp
	ss_msg.ProtoType = ss.SS_PROTO_TYPE_LOGIN_RSP
	body := new(ss.SSMsg_LoginRsp)
	body.LoginRsp = new(ss.MsgLoginRsp)
	prsp := body.LoginRsp
	prsp.CKey = preq.CKey
	prsp.Name = preq.Name
	ss_msg.MsgBody = body

	//do while 0
	for  {
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
		online_logic := -1; //not online
		if sm["online_logic"] != "" {
			online_logic , err = strconv.Atoi(sm["online_logic"]);
			if err != nil {
				log.Err("%s conv online-logic failed! err:%v" , _func_ , err);
				online_logic = 0;
			}
		}
		log.Debug("%s get pass:%s uid:%s online:%d", _func_, pass, uid , online_logic);
		//check pass
		if preq.GetPass() != pass {
			log.Info("%s pass not matched! user:%s c_key:%v ", _func_, preq.Name, preq.CKey)
			prsp.Result = ss.USER_LOGIN_RET_LOGIN_PASS
			break
		}

		//check online
		//already login at other logic should kick first
		if online_logic >= 0 && online_logic != from_serv {
			log.Info("%s user:%s login at other logic server %s kick first!" , _func_ , preq.Name , preq.CKey);
			prsp.Result = ss.USER_LOGIN_RET_LOGIN_MULTI_ON;
			prsp.OnlineLogic = int32(online_logic);
			break;
		}



		//sucess. update online info
		log.Debug("%s pass matched! update online info!", _func_)
		tab_arg := fmt.Sprintf(FORMAT_TAB_USER_GLOBAL , preq.Name);
		pconfig.RedisClient.RedisExeCmd(pconfig.Comm , cb_update_online , append(cb_arg, uid) ,
			"HSET" , tab_arg , "online_logic" , from_serv);
		return
	}

	/*Back to Client*/
	//pack
	buff, err := ss.Pack(&ss_msg)
	if err != nil {
		log.Err("%s pack failed! err:%v", _func_, err)
		return
	}

	//send
	ok = SendToServer(pconfig, buff, from_serv)
	if !ok {
		log.Err("%s send back to %d failed!", _func_, from_serv)
		return
	}
	log.Err("%s send back to %d success!", _func_, from_serv)
	return
}

//cb_arg={0:preq 1:from_server 2:uid}
func cb_update_online(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_update_online>";
	log := comm_config.Log;
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

	uid, ok := cb_arg[2].(string)
	if !ok {
		log.Err("%s conv uid failed! cb:%v", _func_, cb_arg)
		return
	}

	//check error
	if err , ok := result.(error); ok {
		log.Err("%s reply error! name:%s err:%v" , _func_ , preq.Name , err);
		return;
	}

    /*Get Result*/
    ret_code , err := comm.Conv2Int(result);
    if err != nil {
    	log.Err("%s conv result failed! name:%s err:%v" , _func_ , preq.Name , err);
    	return;
	}
    log.Debug("%s ret_code:%d name:%s" , _func_ , ret_code , preq.Name);

    /*Get UserDetail*/
    cmd_arg := fmt.Sprintf(FORMAT_TAB_USER_INFO_REFIX+"%s" , uid);
    pconfig.RedisClient.RedisExeCmd(pconfig.Comm , cb_user_login_get_info , cb_arg , "HGETALL" , cmd_arg);
    return;
}


//cb_arg={0:preq 1:from_server}
func cb_user_login_get_info(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_user_login_get_info>"
	var ss_msg ss.SSMsg
	log := comm_config.Log
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

	log.Debug("%s user:%s pass:%s c_key:%v", _func_, preq.GetName(), preq.GetPass(), preq.GetCKey())
	//check error
	if err , ok := result.(error); ok {
		log.Err("%s reply error! name:%s err:%v" , _func_ , preq.Name , err);
		return;
	}

	/*create rsp */
	ss_msg.ProtoType = ss.SS_PROTO_TYPE_LOGIN_RSP
	body := new(ss.SSMsg_LoginRsp)
	body.LoginRsp = new(ss.MsgLoginRsp)
	prsp := body.LoginRsp
	prsp.CKey = preq.CKey
	prsp.Name = preq.Name
	ss_msg.MsgBody = body

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
			return
		}

		log.Debug("%s get user_Info success! user:%s detail:%v", _func_, preq.Name, sm)
		//Get User Info
		prsp.UserInfo = new(ss.UserInfo)
		prsp.UserInfo.BasicInfo = new(ss.UserBasic)
		pbasic := prsp.UserInfo.BasicInfo
		pbasic.Name = preq.Name

		//basic info(and default)
		var uid int64
		var age int
		var sex int = 1; //1:male 2:female
		var addr string = "moon";
		var level int = 1;
		var puser_blob = new(ss.UserBlob);
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

		//age
		if v, ok := sm["age"]; ok {
			age, err = strconv.Atoi(v)
			if err != nil {
				log.Err("%s conv age failed! err:%v age:%s", _func_, err, v)
			}
		}

		//sex
		if v, ok := sm["sex"]; ok {
			sex, err = strconv.Atoi(v)
			if err != nil {
				log.Err("%s conv sex failed! err:%v sex:%s", _func_, err, v)
			}
		}

		//addr
		if v, ok := sm["addr"]; ok {
			addr = v
		}

		//level
		if v , ok := sm["level"]; ok {
			level , err = strconv.Atoi(v);
			if err != nil {
				log.Err("%s conv level failed! err:%v level:%s" , _func_ , err , v);
				prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR;
				break;
			}
		}

		//blob info
        if v , ok := sm["blob_info"]; ok {
            err = ss.UnPack([]byte(v) , puser_blob);
            if err != nil {
            	log.Err("%s unpack user_blob failed! err:%v uid:%d" , _func_ , err , uid);
            	prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR;
            	break;
			}
		} else { //set blob_info default value
			puser_blob.Exp = 100;
		}



		//Fullfill
		prsp.Result = ss.USER_LOGIN_RET_LOGIN_SUCCESS
		pbasic.Addr = addr
		pbasic.Uid = uid
		pbasic.Age = int32(age)
		pbasic.Level = int32(level);
		prsp.UserInfo.BlobInfo = puser_blob;
		if sex == 1 {
			pbasic.Sex = true //male; false:female
		}
		log.Debug("%s success! user:%s uid:%v", _func_, pbasic.Name, pbasic.Uid)
		break
	}

	/*Back to Client*/
	//pack
	buff, err := ss.Pack(&ss_msg)
	if err != nil {
		log.Err("%s pack failed! err:%v", _func_, err)
		return
	}

	//send
	ok = SendToServer(pconfig, buff, from_serv)
	if !ok {
		log.Err("%s send back to %d failed!", _func_, from_serv)
		return
	}
	log.Err("%s send back to %d success!", _func_, from_serv)
	return
}

//
//cb_arg := []interface{}{from , Uid , Reason};
func cb_update_online_logout(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_update_online_logout>";
	log := comm_config.Log;


	//Get Result
	if err , ok := result.(error); ok {
		log.Err("%s failed! err:%v uid:%v reason:%v" , _func_ , err , cb_arg[1] , cb_arg[2]);
		return;
	}

    log.Info("%s done! ret:%v uid:%v reason:%v" , _func_ , result , cb_arg[1] , cb_arg[2]);
    return;
}

//cb_arg := []interface{}{from , Uid , Reason};
func cb_save_user_logout(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_save_user_logout>";
	log := comm_config.Log;


	//Get Result
	if err , ok := result.(error); ok {
		log.Err("%s failed! err:%v uid:%v reason:%v" , _func_ , err , cb_arg[1] , cb_arg[2]);
		return;
	}

	log.Info("%s done! ret:%v uid:%v reason:%v" , _func_ , result , cb_arg[1] , cb_arg[2]);
	return;
}