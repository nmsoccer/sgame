package lib

//send to logic server
func SendToLogic(pconfig *Config , buff []byte) bool {
	var _func_ = "<SendToLogic>";
	log := pconfig.Comm.Log;
	proc := pconfig.Comm.Proc;
	
	//send
	ret := proc.Send(pconfig.FileConfig.LogicServ, buff , len(buff));
	if ret < 0 {
		log.Err("%s to %d failed! ret:%d" , _func_ , pconfig.FileConfig.LogicServ , ret);
		return false;
	}
	//log.Debug("%s to %d success!" , _func_ , pconfig.FileConfig.LogicServ);
	return true;
}