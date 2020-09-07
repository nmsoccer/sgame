package lib

func SendToServ(pconfig *Config  , target_serv int , buff []byte ) bool {
	var _func_ = "<SendToServ>";
	log := pconfig.Comm.Log;
	proc := pconfig.Comm.Proc;

	ret := proc.Send(target_serv, buff , len(buff));
	if ret < 0 {
		log.Err("%s to %d failed! ret:%d" , _func_ , target_serv , ret);
		return false;
	}
	log.Debug("%s send to %d success!" , _func_ , target_serv);
	return true
}

