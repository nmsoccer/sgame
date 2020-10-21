package lib

import (
	"math/rand"
	"sgame/servers/comm"
)

/*
* ##tables desc##
* #users:global:[name]  <hash> name | pass | uid | salt
* #user:[uid] <hash>  uid | name | age | sex  | addr | level | online_logic | blob_info
* #user:login_lock:[uid] <string> valid_second
* #global:uid <string>
 */

const (
	PASSWD_SALT_LEN                   = 32
	LOGIN_LOCK_LIFE                   = 20                 //login lock life (second)
	FORMAT_TAB_USER_GLOBAL            = "users:global:%s"  //users:global:[name]  ++ hash ++ name | pass | uid | salt
	FORMAT_TAB_USER_INFO_REFIX        = "user:"            // user:[uid] ++ hash ++ uid | name | age | sex  | addr | level | online_logic | blob_info
	FORMAT_TAB_USER_LOGIN_LOCK_PREFIX = "user:login_lock:" //user:login:[uid] <string> valid_second
	FORMAT_TAB_GLOBAL_UID             = "global:uid"       // ++ string

	//Useful FIELD
	FIELD_USER_INFO_ONLINE_LOGIC = "online_logic"

	//REDIS METHOD
	REDIS_METHOD_SINGLE = 0 //simplest method ,only use addr[0] as read and write candidate
	REDIS_METHOD_SM     = 1 //1 master and many slaves. ps:addr[0] is defined as master

	//REDIS OPERATION
	REDIS_OPT_R  = 1 //only for read operation,may dispatch to slave
	REDIS_OPT_W  = 2 //write operation will only disaptch to master
	REDIS_OPT_RW = REDIS_OPT_R | REDIS_OPT_W
)

type RedisClientInfo struct {
	client *comm.RedisClient
	addr   string
}

//select a proper client to exe cmd
func SelectRedisClient(pconfig *Config, redis_opt int) *comm.RedisClient {
	var _func_ = "<SelectClient>"
	log := pconfig.Comm.Log
	count := len(pconfig.RedisClients)
	method := pconfig.FileConfig.RedisMethod

	//none
	if count <= 0 {
		log.Err("%s no clients!", _func_)
		return nil
	}

	//only one
	if count == 1 {
		if pconfig.RedisClients[0].client.GetConnNum() <= 0 {
			log.Err("%s connection all closed! addr:%s", _func_, pconfig.RedisClients[0].addr)
			return nil
		}
		return pconfig.RedisClients[0].client
	}

	//>1 will check method
	var pclient *comm.RedisClient
	switch method {
	case REDIS_METHOD_SINGLE:
		pclient = pconfig.RedisClients[0].client
		if pclient.GetConnNum() <= 0 {
			log.Err("%s single operaton , but connection all closed! addr:%s", _func_, pconfig.RedisClients[0].addr)
			return nil
		}
		return pclient
	case REDIS_METHOD_SM:
		if redis_opt != REDIS_OPT_R { //only for master
			pclient = pconfig.RedisClients[0].client
			if pclient.GetConnNum() <= 0 {
				log.Err("%s sm operation , but master connection all closed! addr:%s", _func_, pconfig.RedisClients[0].addr)
				return nil
			}
			//log.Debug("%s sm operation , return master:%s" , _func_ , pconfig.RedisClients[0].addr)
			return pclient
		}

		//readonly search master and slaves
		pos := rand.Intn(count)
		if pos == 0 {
			pos++ //first not master
		}
		for i := pos; i < count; i++ {
			pclient = pconfig.RedisClients[i].client
			if pclient.GetConnNum() <= 0 {
				log.Err("%s sm operation , but connection all closed! addr:%s", _func_, pconfig.RedisClients[i].addr)
				continue
			}
			log.Debug("%s sm operation , get [%d]:%s", _func_, i, pconfig.RedisClients[i].addr)
			return pclient
		}

		for i := 0; i < pos; i++ {
			pclient = pconfig.RedisClients[i].client
			if pclient.GetConnNum() <= 0 {
				log.Err("%s sm operation , but connection all closed! addr:%s", _func_, pconfig.RedisClients[i].addr)
				continue
			}
			log.Debug("%s sm operation , get [%d]:%s", _func_, i, pconfig.RedisClients[i].addr)
			return pclient
		}
		log.Err("%s sm operation , but all client connection all closed!", _func_)
		return nil
	default:
		log.Err("%s illegal operation:%d", _func_, redis_opt)
	}

	return nil
}

//open redis client
func OpenRedis(pconfig *Config) bool {
	var _func_ = "<OpenRedisAddr>"
	log := pconfig.Comm.Log
	count := len(pconfig.FileConfig.RedisAddr)

	if count <= 0 {
		log.Err("%s failed! empty redis addr!", _func_)
		return false
	}

	pconfig.RedisClients = make([]*RedisClientInfo, count)
	//each client
	for i := 0; i < count; i++ {
		pinfo := new(RedisClientInfo)
		pinfo.addr = pconfig.FileConfig.RedisAddr[i]
		pclient := comm.NewRedisClient(pconfig.Comm, pconfig.FileConfig.RedisAddr[i], pconfig.FileConfig.AuthPass,
			pconfig.FileConfig.MaxConn, pconfig.FileConfig.NormalConn)

		if pclient == nil {
			log.Err("%s failed for new redis client! addr:%s", _func_, pinfo.addr)
			return false
		}
		pinfo.client = pclient
		pconfig.RedisClients[i] = pinfo
	}

	return true
}

//close redis client
func CloseRedis(pconfig *Config) {
	var _func_ = "<CloseRedis>"
	log := pconfig.Comm.Log
	count := len(pconfig.RedisClients)
	defer func() {
		if err := recover(); err != nil {
			log.Fatal("%s panic! err:%v", _func_, err)
		}
	}()

	if count <= 0 {
		return
	}

	for i := 0; i < count; i++ {
		if pconfig.RedisClients[i] != nil {
			log.Info("%s close redis:%s", _func_, pconfig.RedisClients[i].addr)
			pconfig.RedisClients[i].client.Close()
		}
	}

}

//all conn
func CalcRedisConn(pconfig *Config) int {
	conn_count := 0
	for i := 0; i < len(pconfig.RedisClients); i++ {
		conn_count += pconfig.RedisClients[i].client.GetConnNum()
	}

	return conn_count
}

//init db info when first started only use addr[0] as master
func InitRedisDb(arg interface{}) {
	var _func_ = "<InitRedisDb>"
	pconfig, ok := arg.(*Config)
	if !ok {
		return
	}
	log := pconfig.Comm.Log
	if len(pconfig.RedisClients) == 0 {
		log.Info("%s redis not open or client not inited!", _func_)
		return
	}

	log.Info("%s starts...", _func_)
	//init global uid
	pclient := pconfig.RedisClients[0].client
	pclient.RedisExeCmd(pconfig.Comm, cb_init_global_uid, nil, "SETNX",
		FORMAT_TAB_GLOBAL_UID, pconfig.FileConfig.InitUid)
	return
}

//ResetRedis must ensure pconfig.RedisClients all member not nil!
func ResetRedis(pconfig *Config, old_config *FileConfig, new_config *FileConfig) {
	var _func_ = "<ResetRedis>"
	log := pconfig.Comm.Log

	var new_addr string = ""
	var new_auth string = ""
	var new_max int = 0
	var new_normal int = 0
	var reset = false

	for i := 0; i < len(new_config.RedisAddr); i++ {
		//check should reset
		if old_config.RedisAddr[i] != new_config.RedisAddr[i] {
			new_addr = new_config.RedisAddr[i]
			reset = true
		}

		if old_config.AuthPass != new_config.AuthPass {
			new_auth = new_config.AuthPass
			reset = true
		}

		if old_config.MaxConn != new_config.MaxConn {
			new_max = new_config.MaxConn
			reset = true
		}

		if old_config.NormalConn != new_config.NormalConn {
			new_normal = new_config.NormalConn
			reset = true
		}

		if reset && pconfig.RedisClients[i] != nil {
			log.Info("%s will reset redis attr!", _func_)
			pconfig.RedisClients[i].client.Reset(new_addr, new_auth, new_max, new_normal)
			return
		}
	}

	log.Info("%s nothing to do", _func_)
}

/*----------------static func--------------------*/
func cb_init_global_uid(pcomm_config *comm.CommConfig, result interface{}, cb_arg []interface{}) {
	var _func_ = "<cb_init_global_uid>"
	log := pcomm_config.Log

	//check error
	if err, ok := result.(error); ok {
		log.Err("%s failed! err:%v", _func_, err)
		return
	}

	//print
	log.Info("%s result:%v", _func_, result)
	return
}
