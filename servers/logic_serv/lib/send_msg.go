package lib

import "sgame/servers/comm"

//send to connect server
func SendToConnect(pconfig *Config, buff []byte) bool {
	var _func_ = "<SendToConnect>"
	log := pconfig.Comm.Log
	proc := pconfig.Comm.Proc

	//select db server
	target_id := pconfig.FileConfig.ConnServ

	//send
	ret := proc.Send(target_id, buff, len(buff))
	if ret < 0 {
		log.Err("%s to %d failed! ret:%d", _func_, target_id, ret)
		return false
	}
	//log.Debug("%s to %d success!" , _func_ , target_id);
	return true
}

//send to db server
func SendToDb(pconfig *Config, buff []byte) bool {
	var _func_ = "<SendToDb>"
	log := pconfig.Comm.Log
	proc := pconfig.Comm.Proc

	//to db server
	target_id := pconfig.FileConfig.DbServ

	//send
	ret := proc.Send(target_id, buff, len(buff))
	if ret < 0 {
		log.Err("%s to %d failed! ret:%d", _func_, target_id, ret)
		return false
	}
	//log.Debug("%s to %d success!" , _func_ , target_id);
	return true
}

//send to disp hash
func SendToDispHash(pconfig *Config, hash_v int64, buff []byte) bool {
	var _func_ = "<SendToDispHash>"
	log := pconfig.Comm.Log
	proc := pconfig.Comm.Proc

	//SELECT DISP
	disp_serv := -1
	disp_serv = comm.SelectProperServ(pconfig.Comm , comm.SELECT_METHOD_HASH , hash_v , pconfig.FileConfig.DispServList ,
		pconfig.Comm.PeerStats , comm.PERIOD_HEART_BEAT_DEFAULT/1000)
	if disp_serv <= 0 {
		log.Err("%s fail! no proper disp by hash found! key:%d candidate:%v", _func_, hash_v, pconfig.FileConfig.DispServList)
	}

	//BY RAND
	disp_serv = comm.SelectProperServ(pconfig.Comm , comm.SELECT_METHOD_RAND , 0 , pconfig.FileConfig.DispServList ,
		pconfig.Comm.PeerStats , comm.PERIOD_HEART_BEAT_DEFAULT/1000)
	if disp_serv <= 0 {
		log.Err("%s fail! no proper disp by rand found! key:%d candidate:%v", _func_, hash_v, pconfig.FileConfig.DispServList)
		return false
	}


	//Send
	ret := proc.Send(disp_serv, buff, len(buff))
	if ret < 0 {
		log.Err("%s send to %d failed!", _func_, disp_serv)
		return false
	}
	log.Debug("%s to %d success!", _func_, disp_serv)
	return true
}
