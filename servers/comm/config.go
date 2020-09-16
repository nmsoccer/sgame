package comm

import (
	md52 "crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"sgame/lib/log"
	"sgame/lib/proc"
	"syscall"
	"time"
)

const (
	TIME_EPOLL_BASE int64 = 1577808000 //2020-01-01 00:00:00

	TIME_FORMAT_SEC  = "2006-01-02 15:04:05"
	TIME_FORMAT_MILL = "2006-01-02 15:04:05.000"
	TIME_FORMAT_MICR = "2006-01-02 15:04:05.000000"
	TIME_FORMAT_NANO = "2006-01-02 15:04:05.000000000"

	DEFAULT_SERVER_SLEEP_IDLE = 5 //ms. server sleeps when idle

	INFO_EXIT       = 0 //0 server exit #sig-int
	INFO_RELOAD_CFG = 1 //1 server reload config #sig-usr1
	INFO_USR2       = 2 //2 sig-usr2 #sig-usr2
	INFO_PPROF      = 3 //3 go pprof #sig-term

	SELECT_METHOD_RAND = 1 //select id by rand
	SELECT_METHOD_HASH = 2 //select by hash

	RAND_STR_POOL = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ012345678!@#$%^&*()-_=+?<>;:"
)

type CommConfig struct {
	StartTs        int64
	Log            log.LogHeader
	LockFile       *os.File
	Proc           proc.ProcHeader
	ChSig          chan os.Signal
	ChInfo         chan int
	PeerStats      map[int]int64 //peer [procid]->heart_beat_ts
	TickPool       *TickPool
	ServerCfg      interface{} //server *config if assigend
	PProf          ProfileConfig
	ReportCmdToken int64 //if exe report cmd
	ReportCmd      string
}

func InitCommConfig(log_file string, name_space string, proc_id int) *CommConfig {
	pconfig := new(CommConfig)
	if pconfig == nil {
		fmt.Printf("InitCommConfig failed!\n")
		return nil
	}

	//start
	pconfig.StartTs = time.Now().Unix()
	rand.Seed(pconfig.StartTs)
	//log
	lp := log.OpenLog(log_file, log.LOG_DEFAULT_FILT_LEVEL, log.LOG_DEFAULT_DEGREE, log.LOG_DEFAULT_ROTATE,
		log.LOG_DEFAULT_SIZE)
	if lp == nil {
		fmt.Printf("open log %s failed!\n", log_file)
		return nil
	}
	pconfig.Log = lp

	//open bridge
	if proc_id > 0 {
		p := proc.Open(name_space, proc_id)
		if p == nil {
			lp.Err("open bridge <%s:%d> failed!", name_space, proc_id)
			return nil
		}
		pconfig.Proc = p
		lp.Info("open proc bridge <%s:%d> success!", name_space, proc_id)
	}

	//signal
	pconfig.ChSig = make(chan os.Signal, 16)
	signal.Notify(pconfig.ChSig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)

	//msg
	pconfig.ChInfo = make(chan int, 16)

	//peer stats
	pconfig.PeerStats = make(map[int]int64)

	//tick pool
	pconfig.TickPool = NewTickPool(pconfig)
	if pconfig.TickPool == nil {
		lp.Err("new tick pool failed!")
		return nil
	}

	return pconfig
}

func LoadJsonFile(config_file string, file_config interface{}, pconfig *CommConfig) bool {
	var _func_ = "<LoadJsonFile>"
	var log log.LogHeader = nil
	if pconfig != nil {
		log = pconfig.Log
	}

	file, err := os.Open(config_file)
	if err != nil {
		fmt.Printf("%s open %s failed! err:%v\n", _func_, config_file, err)
		if log != nil {
			log.Err("%s open %s failed! err:%v", _func_, config_file, err)
		}
		return false
	}
	defer file.Close()

	//decoder
	var decoder *json.Decoder
	decoder = json.NewDecoder(file)
	if decoder == nil {
		fmt.Printf("%s new json decoder %s failed!\n", _func_, config_file)
		if log != nil {
			log.Err("%s new json decoder %s failed!", _func_, config_file)
		}
		return false
	}

	//decode
	err = decoder.Decode(file_config)
	if err != nil {
		fmt.Printf("%s decode config failed! err:%v\n", _func_, err)
		if log != nil {
			log.Err("%s decode config failed! err:%v", _func_, err)
		}
		return false
	}
	fmt.Printf("FileConfig:%v\n", file_config)
	if log != nil {
		log.Info("%s load %s success!config:%v", _func_, config_file, file_config)
	}
	return true
}

//generate local id
//var seq uint16 = 1;
func GenerateLocalId(wid int16, seq *uint16) int64 {
	var id int64 = 0
	curr_ts := time.Now().Unix() //
	diff := curr_ts - TIME_EPOLL_BASE

	*seq = *seq + 1
	if *seq >= 65530 {
		*seq = 1
	}

	id = ((int64(*seq) & 0xFFFF) << 47) | ((int64(wid) & 0xFFFF) << 31) | (diff & 0x7FFFFFFF)
	return id
}

/*select a proper id
* @method:refer SELECT_METHOD_XX
* @hash_v:if method==SELECT_METHOD_HASH ,it sets hash value
* @candidate:: candidate proc_id of servers
* @stats: server heart stats
* @life_time: valid life_time in stats
* @result <=0 failed >0 success
* PS:if candidate lens == 1 will only select one if in life_time
 */
func SelectProperServ(pconfig *CommConfig, method int, hash_v int64, candidate []int, stats map[int]int64, life_time int64) int {
	var _func_ = "<SelectProperServ>"
	var serv_id int
	log := pconfig.Log
	curr_ts := time.Now().Unix()

	//empty
	if len(candidate) <= 0 {
		log.Err("%s fail! candidate empty!", _func_)
		return -1
	}

	//only one member
	if len(candidate) == 1 {
		serv_id = candidate[0]
		last_heart, ok := stats[serv_id]
		if !ok {
			log.Err("%s fail! no heart found of %d", _func_, serv_id)
			return -1
		}

		if (life_time + last_heart) < curr_ts {
			log.Err("%s fail! heart expired! last_heart:%d now:%d", _func_, last_heart, curr_ts)
			return -1
		}

		return candidate[0]
	}

	//more than one member
	switch method {
	case SELECT_METHOD_RAND:
		//rand one
		total := len(candidate)
		i := rand.Intn(len(candidate))
		count := 0

		for {
			if count >= total {
				break
			}

			//search
			serv_id = candidate[i]
			if stats[serv_id]+life_time >= curr_ts {
				return serv_id
			}

			//iter again
			i++
			i = i % total
			count++
		}
		log.Err("%s [rand] no proper serv  found!", _func_)
	case SELECT_METHOD_HASH:
		if hash_v <= 0 {
			log.Err("%s [hash] hash_v:%d illegal!", _func_, hash_v)
			break
		}

		pos := hash_v % int64(len(candidate))
		serv_id = candidate[pos]
		//alive
		if stats[serv_id]+life_time >= curr_ts {
			return serv_id
		}
		log.Err("%s [hash] no proper serv  found!", _func_)
	default:
		log.Err("%s fail! illegal method:%d", _func_, method)

	}

	return -1
}

//generate rand str lenth==str_len
func GenRandStr(str_len int) (string, error) {
	if str_len > 1000 {
		return "", errors.New("length too long!")
	}

	b := make([]byte, str_len)
	pool_len := len(RAND_STR_POOL)
	pos := 0
	for i := 0; i < str_len; i++ {
		pos = rand.Intn(pool_len)
		b[i] = RAND_STR_POOL[pos]
	}

	return string(b), nil
}

//enc password
func EncPassString(pass string, salt string) string {
	md5 := md52.New()
	md5.Write([]byte(pass))
	md5.Write([]byte(salt))
	res := md5.Sum(nil)
	return hex.EncodeToString(res)
}

/*Read All File content
@close:if close file after reading
 */
func ReadFile(file_name string , close bool) ([]byte , error) {
	//open file
	file , err := os.Open(file_name)
	if err != nil {
		return nil , err
	}
	defer func() {
		if close {
			file.Close()
		}
	}()

	//read all
	b , err := ioutil.ReadAll(file)
	if err != nil {
		return nil , err
	}

	return b , nil
}
