package lib

import (
  "sgame/proto/cs"
)

func SendLoginReq(pconfig *Config , client_key int64 , plogin_req *cs.CSLoginReq) {
	var _func_ = "<SendLoginReq>";
	log := pconfig.Comm.Log;
	
	//send
	log.Debug("%s send login pkg to logic! user:%s device:%s" , _func_ , plogin_req.Name ,plogin_req.Device);
	
}