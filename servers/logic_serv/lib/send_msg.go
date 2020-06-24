package lib


//send to connect server
func SendToConnect(pconfig *Config , buff []byte) bool {
	var _func_ = "<SendToConnect>";
	log := pconfig.Comm.Log;
	proc := pconfig.Comm.Proc;
	
	//select db server
	target_id := pconfig.FileConfig.ConnServ;
	
	//send
	ret := proc.Send(target_id , buff , len(buff));
	if ret < 0 {
		log.Err("%s to %d failed! ret:%d" , _func_ , target_id , ret);
		return false;
	}
	log.Debug("%s to %d success!" , _func_ , target_id);
	return true;
}



//send to db server
func SendToDb(pconfig *Config , buff []byte) bool {
	var _func_ = "<SendToDb>";
	log := pconfig.Comm.Log;
	proc := pconfig.Comm.Proc;
	
	//select db server
	/*
	stats := pconfig.Comm.PeerStats;
	
	db_id := pconfig.FileConfig.MasterDb;*/
	target_id := pconfig.FileConfig.MasterDb;
	
	//send
	ret := proc.Send(target_id , buff , len(buff));
	if ret < 0 {
		log.Err("%s to %d failed! ret:%d" , _func_ , target_id , ret);
		return false;
	}
	log.Debug("%s to %d success!" , _func_ , target_id);
	return true;
}