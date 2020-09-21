package lib

import (
	"sgame/servers/comm"
)

/*
* ##tables desc##
* #users:global:[name]  <hash> name | pass | uid | salt
* #user:[uid] <hash>  uid | name | age | sex  | addr | level | online_logic | blob_info
* #user:login_lock:[uid] <string> valid_second
* #global:uid <string>
 */


const (
	PASSWD_SALT_LEN = 32
	LOGIN_LOCK_LIFE = 20 //login lock life (second)
	FORMAT_TAB_USER_GLOBAL="users:global:%s" //users:global:[name]  ++ hash ++ name | pass | uid | salt
	FORMAT_TAB_USER_INFO_REFIX="user:" // user:[uid] ++ hash ++ uid | name | age | sex  | addr | level | online_logic | blob_info
	FORMAT_TAB_USER_LOGIN_LOCK_PREFIX="user:login_lock:" //user:login:[uid] <string> valid_second
	FORMAT_TAB_GLOBAL_UID="global:uid" // ++ string

	//Useful FIELD
	FIELD_USER_INFO_ONLINE_LOGIC = "online_logic"

)




func OpenRedis(pconfig *Config) *comm.RedisClient{
	var _func_ = "<OpenRedisAddr>";
		
	log := pconfig.Comm.Log;
	pclient := comm.NewRedisClient(pconfig.Comm , pconfig.FileConfig.RedisAddr , pconfig.FileConfig.AuthPass , 
		pconfig.FileConfig.MaxConn , pconfig.FileConfig.NormalConn);
	
	if pclient == nil {
	    log.Err("%s fail!" , _func_);
	    return nil;	
	}
	
	return pclient;
}


//init db info when first started
func InitRedisDb(arg interface{}) {
    var _func_ = "<InitRedisDb>";
    pconfig , ok := arg.(*Config);
    if !ok {
    	return;
	}
	log := pconfig.Comm.Log;
	if pconfig.RedisClient == nil {
		log.Info("%s redis not open or client not inited!" , _func_);
		return;
	}

    log.Info("%s starts..." , _func_);
    //init global uid
    pconfig.RedisClient.RedisExeCmd(pconfig.Comm , cb_init_global_uid , nil , "SETNX" ,
    	FORMAT_TAB_GLOBAL_UID , pconfig.FileConfig.InitUid);

    return;
}

//ResetRedis
func ResetRedis(pconfig *Config , old_config *FileConfig, new_config *FileConfig) {
	var _func_ = "<ResetRedis>"
	log := pconfig.Comm.Log

	var new_addr string = ""
	var new_auth string = ""
	var new_max  int = 0
	var new_normal int = 0
    var reset = false

	//check should reset
	if old_config.RedisAddr != new_config.RedisAddr {
		new_addr = new_config.RedisAddr
		reset = true
	}

    if old_config.AuthPass != new_config.AuthPass {
    	new_auth = new_config.AuthPass
    	reset = true
	}

    if old_config.MaxConn != new_config.MaxConn {
    	new_max = new_config.MaxConn
    	reset = true
	}

    if old_config.NormalConn != new_config.NormalConn {
    	new_normal = new_config.NormalConn
    	reset = true
	}

	if reset {
		log.Info("%s will reset redis attr!" , _func_)
		pconfig.RedisClient.Reset(new_addr , new_auth , new_max , new_normal)
		return
	}

	log.Info("%s nothing to do" , _func_)
}



/*----------------static func--------------------*/
func cb_init_global_uid(pcomm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
    var _func_ = "<cb_init_global_uid>";
    log := pcomm_config.Log;

    //check error
    if err , ok := result.(error); ok {
    	log.Err("%s failed! err:%v" , _func_ , err);
    	return;
	}

	//print
	log.Info("%s result:%v" , _func_ , result);
    return;
}