package lib

import (
	"fmt"
	"sgame/proto/ss"
	"sgame/servers/comm"
)

func RecvRegReq(pconfig *Config, preq *ss.MsgRegReq, from int) {
	var _func_ = "<RecvRegReq>"
	log := pconfig.Comm.Log
	log.Info("%s user:%s addr:%s sex:%v from:%d role_name:%s", _func_, preq.Name, preq.Addr, preq.Sex, from , preq.RoleName)

	//common Check
	if preq.RoleName == "" {
		preq.RoleName = preq.Name
	}

	//set name
	tab_name := fmt.Sprintf(FORMAT_TAB_USER_GLOBAL, preq.Name)
	pconfig.RedisClient.RedisExeCmd(pconfig.Comm, cb_set_global_name, []interface{}{preq, from}, "HSETNX",
		tab_name, "name", preq.Name)

	return
}

func SendRegRsp(pconfig *Config, preq *ss.MsgRegReq, target_serv int, result ss.REG_RESULT) {
	var _func_ = "<SendRegRsp>"
	log := pconfig.Comm.Log

	log.Info("%s name:%s target:%d result:%v", _func_, preq.Name, target_serv, result)
	//msg
	var ss_msg ss.SSMsg
	pRegRsp := new(ss.MsgRegRsp)
	pRegRsp.Name = preq.Name
	pRegRsp.CKey = preq.CKey
	pRegRsp.Result = result

	//fill
	err := comm.FillSSPkg(&ss_msg , ss.SS_PROTO_TYPE_REG_RSP , pRegRsp)
	if err != nil {
		log.Err("%s pack failed! err:%v", _func_, err)
		return
	}

	//sendback
	if ok := SendToServ(pconfig, target_serv , &ss_msg); !ok {
		log.Err("%s send to logic:%d failed!", _func_, target_serv)
	}
}

/*-----------------------static func-----------------------*/
//cb_arg=[preq , from]
func cb_set_global_name(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_set_global_name>"
	log := comm_config.Log

	/*---------mostly common logic--------------*/
	//get config
	pconfig, ok := comm_config.ServerCfg.(*Config)
	if !ok {
		log.Err("%s failed! convert config fail!", _func_)
		return
	}

	//conv arg
	preq, ok := cb_arg[0].(*ss.MsgRegReq)
	if !ok {
		log.Err("%s conv req failed!", _func_)
		return
	}

	from, ok := cb_arg[1].(int)
	if !ok {
		log.Err("%s conv from failed! name:%s", _func_, preq.Name)
		return
	}

	/*---------result handle--------------*/
	//check result
	err, ok := result.(error)
	if ok {
		log.Err("%s exe failed! err:%v name:%s", _func_, err, preq.Name)
		SendRegRsp(pconfig, preq, from, ss.REG_RESULT_REG_DB_ERR)
		return
	}

	//get result
	ret, err := comm.Conv2Int(result)
	if err != nil {
		log.Err("%s conv result failed! err:%v name:%s", _func_, err, preq.Name)
		SendRegRsp(pconfig, preq, from, ss.REG_RESULT_REG_DB_ERR)
		return
	}

	//check ret
	if ret == 0 {
		log.Info("%s name exists!name:%s", _func_, preq.Name)
		SendRegRsp(pconfig, preq, from, ss.REG_RESULT_REG_DUP_NAME)
		return
	}

	//alloc uid
	log.Info("%s ret:%d try to alloc uid. name:%s", _func_, ret, preq.Name)
	pconfig.RedisClient.RedisExeCmd(pconfig.Comm, cb_alloc_uid, cb_arg, "INCRBY", FORMAT_TAB_GLOBAL_UID, pconfig.FileConfig.UidIncr)
}

func cb_alloc_uid(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_alloc_uid>"
	log := comm_config.Log

	/*---------mostly common logic--------------*/
	//get config
	pconfig, ok := comm_config.ServerCfg.(*Config)
	if !ok {
		log.Err("%s failed! convert config fail!", _func_)
		return
	}

	//conv arg
	preq, ok := cb_arg[0].(*ss.MsgRegReq)
	if !ok {
		log.Err("%s conv req failed!", _func_)
		return
	}

	from, ok := cb_arg[1].(int)
	if !ok {
		log.Err("%s conv from failed! name:%s", _func_, preq.Name)
		return
	}

	/*---------result handle--------------*/
	//check result
	err, ok := result.(error)
	if ok {
		log.Err("%s exe failed! err:%v name:%s", _func_, err, preq.Name)
		SendRegRsp(pconfig, preq, from, ss.REG_RESULT_REG_DB_ERR)
		return
	}

	//get result
	uid, err := comm.Conv2Int64(result)
	if err != nil {
		log.Err("%s conv result failed! err:%v name:%s", _func_, err, preq.Name)
		SendRegRsp(pconfig, preq, from, ss.REG_RESULT_REG_DB_ERR)
		return
	}

	//check uid
	if uid <= pconfig.FileConfig.InitUid {
		log.Info("%s uid illegal! uid:%d name:%s", _func_, uid, preq.Name)
		SendRegRsp(pconfig, preq, from, ss.REG_RESULT_REG_DB_ERR)
		return
	}

	//generate salt
	salt , err := comm.GenRandStr(PASSWD_SALT_LEN)
	if err != nil {
		log.Err("%s generate salt failed! err:%v name:%s" , _func_ , err , preq.Name)
		SendRegRsp(pconfig, preq, from, ss.REG_RESULT_REG_DB_ERR)
		return
	}

	//enc pass
	enc_pass := comm.EncPassString(preq.Pass , salt)

	//set user_global[users:global:name]
	log.Info("%s uid:%d try to set user global. name:%s", _func_, uid, preq.Name)
	tab_name := fmt.Sprintf(FORMAT_TAB_USER_GLOBAL, preq.Name)
	pconfig.RedisClient.RedisExeCmd(pconfig.Comm, cb_set_global_info, append(cb_arg, uid), "HMSET", tab_name, "uid", uid,
		"pass", enc_pass , "salt" , salt)
}

//set global
//cb_arg[preq , from , uid]
func cb_set_global_info(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_set_global_info>"
	log := comm_config.Log

	/*---------mostly common logic--------------*/
	//get config
	pconfig, ok := comm_config.ServerCfg.(*Config)
	if !ok {
		log.Err("%s failed! convert config fail!", _func_)
		return
	}

	//conv arg
	preq, ok := cb_arg[0].(*ss.MsgRegReq)
	if !ok {
		log.Err("%s conv req failed!", _func_)
		return
	}

	from, ok := cb_arg[1].(int)
	if !ok {
		log.Err("%s conv from failed! name:%s", _func_, preq.Name)
		return
	}

	uid, ok := cb_arg[2].(int64)
	if !ok {
		log.Err("%s conv uid failed! name:%s", _func_, preq.Name)
		return
	}

	/*---------result handle--------------*/
	//check result
	err, ok := result.(error)
	if ok {
		log.Err("%s exe failed! err:%v name:%s", _func_, err, preq.Name)
		SendRegRsp(pconfig, preq, from, ss.REG_RESULT_REG_DB_ERR)
		return
	}

	//get result
	res, err := comm.Conv2String(result)
	if err != nil {
		log.Err("%s conv result failed! err:%v name:%s", _func_, err, preq.Name)
		SendRegRsp(pconfig, preq, from, ss.REG_RESULT_REG_DB_ERR)
		return
	}
	//res should always right
	log.Info("%s res:%s name:%s uid:%d will set user info", _func_, res, preq.Name, uid)
	tab_name := fmt.Sprintf(FORMAT_TAB_USER_INFO_REFIX+"%d", uid)
	sex_v := 1
	if preq.Sex == false {
		sex_v = 2
	}
	pconfig.RedisClient.RedisExeCmd(pconfig.Comm, cb_reg_set_user_info, cb_arg, "HMSET", tab_name, "uid", uid,
		"name", preq.RoleName, "addr", preq.Addr, "sex", sex_v , "online_logic", -1)
}

func cb_reg_set_user_info(comm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_reg_set_user_info>"
	log := comm_config.Log

	/*---------mostly common logic--------------*/
	//get config
	pconfig, ok := comm_config.ServerCfg.(*Config)
	if !ok {
		log.Err("%s failed! convert config fail!", _func_)
		return
	}

	//conv arg
	preq, ok := cb_arg[0].(*ss.MsgRegReq)
	if !ok {
		log.Err("%s conv req failed!", _func_)
		return
	}

	from, ok := cb_arg[1].(int)
	if !ok {
		log.Err("%s conv from failed! name:%s", _func_, preq.Name)
		return
	}

	uid, ok := cb_arg[2].(int64)
	if !ok {
		log.Err("%s conv uid failed! name:%s", _func_, preq.Name)
		return
	}

	/*---------result handle--------------*/
	//check result
	err, ok := result.(error)
	if ok {
		log.Err("%s exe failed! err:%v name:%s", _func_, err, preq.Name)
		SendRegRsp(pconfig, preq, from, ss.REG_RESULT_REG_DB_ERR)
		return
	}

	//get result
	res, err := comm.Conv2String(result)
	if err != nil {
		log.Err("%s conv result failed! err:%v name:%s", _func_, err, preq.Name)
		SendRegRsp(pconfig, preq, from, ss.REG_RESULT_REG_DB_ERR)
		return
	}
	//res should always right
	log.Info("%s res:%s name:%s uid:%d success!", _func_, res, preq.Name, uid)
	SendRegRsp(pconfig, preq, from, ss.REG_RESULT_REG_SUCCESS)
}
