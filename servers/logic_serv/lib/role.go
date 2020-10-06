package lib

import "sgame/proto/ss"

func SaveRolesOnExit(pconfig *Config) {
	var _func_ = "<SaveRolesOnExit>"
	log := pconfig.Comm.Log

	//check online
	if pconfig.Users.curr_online <= 0 || pconfig.Users.user_map == nil {
		return
	}

	//each role
	log.Info("%s will kickout online user! online:%d" , _func_ , pconfig.Users.curr_online)
	for uid , _ := range(pconfig.Users.user_map) {
		UserLogout(pconfig , uid , ss.USER_LOGOUT_REASON_LOGOUT_SERVER_SHUT);
		//send to client
		SendLogoutRsp(pconfig , uid , ss.USER_LOGOUT_REASON_LOGOUT_SERVER_SHUT , "server down");
	}
}

