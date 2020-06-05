package lib

import (
  "time"
)


func ReadClients(pconfig *Config) int64{
	var _func_ = "<ReadClients>";
	log := pconfig.Comm.Log;
		
	//get results
	results := pconfig.TcpServ.Recv(pconfig.Comm);
	//log.Debug("%s get results:%v len:%d" , _func_ , results , len(results));
	if results == nil || len(results)==0 {
		return 0;
	}
	
	start_ts := time.Now().UnixNano();	
	//print
	for i:=0; i<len(results); i++ {
		log.Debug("%s read:%s" , _func_ , string(results[i]));
	}
	
	//diff
	diff := time.Now().UnixNano()-start_ts;
	log.Debug("%s cost %dus pkg:%d" , _func_ , diff/1000 , len(results));
	return diff;
}

