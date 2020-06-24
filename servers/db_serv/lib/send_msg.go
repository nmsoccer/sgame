package lib

//send to other server
func SendToServer(pconfig *Config , buff []byte , target_id int) bool {
	var _func_ = "<SendToServer>";
	log := pconfig.Comm.Log;
	proc := pconfig.Comm.Proc;
	
	//send
	ret := proc.Send(target_id , buff , len(buff));
	if ret < 0 {
		log.Err("%s to %d failed! ret:%d" , _func_ , target_id , ret);
		return false;
	}
	log.Debug("%s to %d success!" , _func_ , target_id);
	return true;
}