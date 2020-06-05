package lib

import (
  "sgame/servers/comm" 
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