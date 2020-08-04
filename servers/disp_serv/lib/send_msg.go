package lib


//send to logic server
func SendToLogic(pconfig *Config , logic_serv int  , buff []byte) bool {
	var _func_ = "<SendToLogic>";
	log := pconfig.Comm.Log;
	proc := pconfig.Comm.Proc;

	//check logic
	found := false;
	for _ , id := range pconfig.FileConfig.LogicServList {
		if id == logic_serv {
			found = true;
			break;
		}
	}
	if !found {
		log.Err("%s illegal logic_serv:%d list:%v" , _func_ , logic_serv , pconfig.FileConfig.LogicServList)
		return false;
	}


	//send
	ret := proc.Send(logic_serv, buff , len(buff));
	if ret < 0 {
		log.Err("%s to %d failed! ret:%d" , _func_ , logic_serv , ret);
		return false;
	}
	log.Debug("%s send to %d success!" , _func_ , logic_serv);
	return true;
}