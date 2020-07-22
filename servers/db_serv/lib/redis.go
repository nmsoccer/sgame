package lib

import (
	"sgame/servers/comm"
)

/*
* tables desc
* users:global:[name]  > pass | uid | online_logic
* user:[uid] >  uid | name | age | sex  | addr
 */


const (
	FORMAT_TAB_USER_GLOBAL="users:global:%s" //users:global:[name]  ++ hash ++ name | pass | uid | online_logic
	FORMAT_TAB_USER_INFO_REFIX="user:" // user:[uid] ++ hash ++ uid | name | age | sex  | addr | level | blob_info
	FORMAT_TAB_GLOBAL_UID="global:uid" // ++ string
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