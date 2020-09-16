package lib

import (
	lnet "sgame/lib/net"
	"sgame/servers/comm"
)

//read rsa key files if enc_type==NET_ENCRYPPT_RSA
func ReadRsaKeyFiles(pconfig *Config) bool {
	var _func_ = "<ReadRsaKeyFiles>"
	log := pconfig.Comm.Log

	if pconfig.FileConfig.EncType != lnet.NET_ENCRYPT_RSA {
		return true
	}

	//check key file
	if len(pconfig.FileConfig.RsaPubFile)<=0 || len(pconfig.FileConfig.RsaPriFile)<=0 {
		log.Err("%s failed! EncType is RSA but Key Files not defined! Please Check!" , _func_)
		return false
	}

	//read pub key
	pub_key , err := comm.ReadFile(pconfig.FileConfig.RsaPubFile , true)
	if err != nil {
		log.Err("%s read %s failed! err:%v" , _func_ , pconfig.FileConfig.RsaPubFile , err)
		return false
	}
	log.Debug("%s read %s content:%s" , _func_ , pconfig.FileConfig.RsaPubFile , string(pub_key))

	//read priv key
	pri_key , err := comm.ReadFile(pconfig.FileConfig.RsaPriFile , true)
	if err != nil {
		log.Err("%s read %s failed! err:%v" , _func_ , pconfig.FileConfig.RsaPriFile , err)
		return false
	}
	log.Debug("%s read %s content:%s" , _func_ , pconfig.FileConfig.RsaPriFile , string(pri_key))

	//set
	pconfig.RsaPubKey = pub_key
	pconfig.RsaPriKey = pri_key
	return true
}