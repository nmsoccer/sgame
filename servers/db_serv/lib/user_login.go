package lib

import (
	"sgame/proto/ss"
	"sgame/servers/comm"
	"strconv"
)

/*
* tables desc
* users:global:[name]  > pass | uid
* user:[uid] >  uid | name | age | sex  | addr  
*/

func RecvUserLoginReq(pconfig *Config , preq *ss.MsgLoginReq , from int) {
	var _func_ = "<RecvUserLoginReq>";
	log := pconfig.Comm.Log;
	
	log.Debug("%s user:%s pass:%s c_key:%v" , _func_ , preq.GetName() , preq.GetPass() , preq.GetCKey());
	//query pass
	pclient := pconfig.RedisClient;
	cmd_arg := "users:global:" + preq.Name;
	pclient.RedisExeCmd(pconfig.Comm , cb_user_login_check_pass , []interface{}{pconfig , preq , from} , 
	"HGETALL" , cmd_arg);
}

/*---------------------------------STATIC FUNC-----------------------------*/
func cb_user_login_check_pass(comm_config *comm.CommConfig , result interface{} , cb_arg []interface{}) {
	var _func_ = "<cb_user_login_check_pass>";
	var ss_msg ss.SSMsg;
	log := comm_config.Log;
	
	//conv cb
	pconfig , ok := cb_arg[0].(*Config);
	if !ok {
		log.Err("%s conv config failed! cb:%v" , _func_ , cb_arg);
		return;
	}
	
	preq , ok := cb_arg[1].(*ss.MsgLoginReq);
	if !ok {
		log.Err("%s conv req failed! cb:%v" , _func_ , cb_arg);
		return;
	}
	
	from_serv , ok := cb_arg[2].(int);
	if !ok {
		log.Err("%s conv from failed! cb:%v" , _func_ , cb_arg);
		return;
	}
	
	log.Debug("%s user:%s pass:%s c_key:%v" , _func_ , preq.GetName() , preq.GetPass() , preq.GetCKey());
	//rsp
	ss_msg.ProtoType = ss.SS_PROTO_TYPE_LOGIN_RSP;
	body := new(ss.SSMsg_LoginRsp);
	body.LoginRsp = new(ss.MsgLoginRsp);
	prsp := body.LoginRsp;
	prsp.CKey = preq.CKey;
	prsp.Name = preq.Name;
	ss_msg.MsgBody = body;
	
	//do while 0
	for i:=0; i<1; i++ {
	    //check result may need reg
	    if result == nil {
		    log.Info("%s no user:%s exist!" , _func_ , preq.Name);
		    prsp.Result = ss.USER_LOGIN_RET_LOGIN_EMPTY;
		    break;
	    }
	        	    	
	    //conv result
	    sm , err := comm.Conv2StringMap(result);
	    if err != nil {
		    log.Err("%s conv result failed! err:%v" , _func_ , err);
		    return;
	    }
	    
	    pass := sm["pass"];
	    uid := sm["uid"];
	    log.Debug("%s get pass:%s uid:%s" , _func_ , pass , uid);	
	    //check pass
	    if preq.GetPass() != pass {
		    log.Info("%s pass not matched! user:%s c_key:%v " , _func_ , preq.Name , preq.CKey);
		    prsp.Result = ss.USER_LOGIN_RET_LOGIN_PASS;
		    break;
	    }
	    
	    //sucess. query user info
	    log.Debug("%s pass matched! query user info!" , _func_);
	    pclient := pconfig.RedisClient;
	    cmd_arg := "user:" + uid;
	    pclient.RedisExeCmd(pconfig.Comm , cb_user_login_get_info , cb_arg , 
	      "HGETALL" , cmd_arg);
	    return;
	}
	
	/*Back to Client*/
	//pack
	buff , err := ss.Pack(&ss_msg);
	if err != nil {
		log.Err("%s pack failed! err:%v" , _func_ , err);
		return;
	}
	
	//send
	ok = SendToServer(pconfig, buff, from_serv);
	if !ok {
		log.Err("%s send back to %d failed!" , _func_ , from_serv);
		return;
	}
	log.Err("%s send back to %d success!" , _func_ , from_serv);
	return;
}


func cb_user_login_get_info(comm_config *comm.CommConfig , result interface{} , cb_arg []interface{}) {
	var _func_ = "<cb_user_login_get_info>";
	var ss_msg ss.SSMsg;
	log := comm_config.Log;
	
	/*convert callback arg*/
	pconfig , ok := cb_arg[0].(*Config);
	if !ok {
		log.Err("%s get config failed!" , _func_);
		return;
	}
	
	preq , ok := cb_arg[1].(*ss.MsgLoginReq);
	if !ok {
		log.Err("%s conv cb failed!" , _func_);
		return;
	}
	
	from_serv , ok := cb_arg[2].(int);
	if !ok {
		log.Err("%s conv from failed! cb:%v" , _func_ , cb_arg);
		return;
	}
	
	log.Debug("%s user:%s pass:%s c_key:%v" , _func_ , preq.GetName() , preq.GetPass() , preq.GetCKey());
	/*create rsp */
	ss_msg.ProtoType = ss.SS_PROTO_TYPE_LOGIN_RSP;
	body := new(ss.SSMsg_LoginRsp);
	body.LoginRsp = new(ss.MsgLoginRsp);
	prsp := body.LoginRsp;
	prsp.CKey = preq.CKey;
	prsp.Name = preq.Name;
	ss_msg.MsgBody = body;
	
	//do while 0
	for i:=0; i<1; i++ {
	    //check result 
	    if result == nil {
		    log.Err("%s no user detail:%s exist!" , _func_ , preq.Name);
		    prsp.Result = ss.USER_LOGIN_RET_LOGIN_EMPTY;
		    break;
	    }
	        	    	
	    //conv result
	    sm , err := comm.Conv2StringMap(result);
	    if err != nil {
		    log.Err("%s conv result failed! err:%v" , _func_ , err);
		    return;
	    }
	    
	    log.Debug("%s get user_Info success! user:%s detail:%v" , _func_ , preq.Name , sm);	    
	    //Get User Info 
	   prsp.UserInfo = new(ss.UserInfo);
	   prsp.UserInfo.BasicInfo = new(ss.UserBasic);
	   pbasic := prsp.UserInfo.BasicInfo;	   
	   pbasic.Name = preq.Name;
	   
	     //uid	   
	   var uid int64;
	   var age int;
	   var sex int;
	   var addr string;
	   if v , ok := sm["uid"]; ok {
	       uid , err = strconv.ParseInt(v, 10, 64);
	       if err != nil {
	       	    log.Err("%s conv uid failed! err:%v uid:%s" , _func_ , err , v);
	       	    prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR;
	       	    break;
	       }
	   } else {
	   	    log.Err("%s no uid found of user:%s" , _func_ , preq.Name);
	   	    prsp.Result = ss.USER_LOGIN_RET_LOGIN_ERR;
	   	    break;
	   }
	   
	       //age
	   if v , ok := sm["age"]; ok {
	       age , err = strconv.Atoi(v);
	       if err != nil {
	       	    log.Err("%s conv age failed! err:%v age:%s" , _func_ , err , v);
	       }	   
	   }
	   
	       //sex
	   if v , ok := sm["sex"]; ok {
	   	    sex , err = strconv.Atoi(v);
	   	    if err != nil {
	   	    	log.Err("%s conv sex failed! err:%v sex:%s" , _func_ , err , v);
	   	    }
	   }
	   
			//addr
		if v , ok := sm["addr"]; ok {
			addr = v;
		}	    
	   
	    //Fullfill
	    prsp.Result = ss.USER_LOGIN_RET_LOGIN_SUCCESS;
	    pbasic.Addr = addr;
	    pbasic.Uid = uid;
	    pbasic.Age = int32(age);
	    if sex == 1 {
	    	pbasic.Sex = true; //male
	    }
	    log.Debug("%s success! user:%s uid:%v" , _func_ , pbasic.Name , pbasic.Uid);	    
	    break;
	}
	
	/*Back to Client*/
	//pack
	buff , err := ss.Pack(&ss_msg);
	if err != nil {
		log.Err("%s pack failed! err:%v" , _func_ , err);
		return;
	}
	
	//send
	ok = SendToServer(pconfig, buff, from_serv);
	if !ok {
		log.Err("%s send back to %d failed!" , _func_ , from_serv);
		return;
	}
	log.Err("%s send back to %d success!" , _func_ , from_serv);
	return;
}
