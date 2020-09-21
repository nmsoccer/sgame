/*
Redis Wrapper of redigo

EXE Redis Two Methods:
1) Asynchronously Using CallBack
RedisClient.RedisExeCmd(pconfig *CommConfig, cb_func RedisCallBack, cb_arg []interface{}, cmd string, arg ...interface{})

2) Synchronise Using SyncCmdHead
constraint:Must In an independant go-routine and Release After Alloc
example:
func RecvXXPackage() {
  go func() {
    SyncCmdHead phead = RedisClient.AllocSyncCmdHead()  //1.
    defer RedisClient.FreeSyncCmdHead(phead)  //2.
    ...
    RedisClient.RedisExeCmdSync(phead , ...)  //3.
  }
}

*/

package comm

import (
	"errors"
	"github.com/gomodule/redigo/redis"
	"sync"
	"time"
)

const (
	HARD_MAX_CONN_LIMIT = 1000
	REDIS_CONN_NONE = 0
	REDIS_CONN_ING = 1
	REDIS_CONN_DONE = 2
)

var last_check int64

const (
	check_conn_circle = 5
	block_queue_len   = (100000) //max block go-routine ...
)

type reset_attr struct {
	addr string
	auth string
	max_count int
	normal_count int
}


type RedisClient struct {
	sync.Mutex
	comm_config *CommConfig
	addr         string
	auth         string
	max_count    int
	normal_count int
	conn_count   int
	conn_stats   []int8       //each conn stat refer REDIS_CONN_XX
	conns        []redis.Conn //each conn
	exit_queue   chan bool    //exit flag
	idle_queue   chan int     // idle index of conn
	err_queue    chan int     //err connection done-->err
	block_queue  chan int8    //block request. cap is unlimited. init:1M
	reset_queue  chan bool    //reset flag
	reset_attr   reset_attr  //reset attr if new setting
}

//redis exe cmd using synchronized
type SyncCmdHead struct {
	conn redis.Conn
	idx  int
}

//call back
type RedisCallBack func(pconfig *CommConfig, result interface{}, cb_arg []interface{})

//Conver Wrapper refer reply.go in goredis
func Conv2Int(result interface{}) (int, error) {
	return redis.Int(result, nil)
}
func Conv2Int64(result interface{}) (int64, error) {
	return redis.Int64(result, nil)
}
func Conv2UInt64(result interface{}) (uint64, error) {
	return redis.Uint64(result, nil)
}
func Conv2Float64(result interface{}) (float64, error) {
	return redis.Float64(result, nil)
}
func Conv2String(result interface{}) (string, error) {
	return redis.String(result, nil)
}
func Conv2Bytes(result interface{}) ([]byte, error) {
	return redis.Bytes(result, nil)
}
func Conv2Values(result interface{}) ([]interface{}, error) {
	return redis.Values(result, nil)
}
func Conv2Strings(result interface{}) ([]string, error) {
	return redis.Strings(result, nil)
}
func Conv2StringMap(result interface{}) (map[string]string, error) {
	return redis.StringMap(result, nil)
}
func Conv2IntMap(result interface{}) (map[string]int, error) {
	return redis.IntMap(result, nil)
}
func Conv2Int64Map(result interface{}) (map[string]int64, error) {
	return redis.Int64Map(result, nil)
}

//New RedisClient
func NewRedisClient(pconfig *CommConfig, redis_addr string, auth string, max_conn, normal_conn int) *RedisClient {
	var _func_ = "<NewRedisClient>"
	log := pconfig.Log

	//new
	pclient := new(RedisClient)
	if pclient == nil {
		log.Err("%s Failed! new fail!", _func_)
		return nil
	}

	//check max
	if max_conn > HARD_MAX_CONN_LIMIT {
		log.Err("%s Failed! max_conn:%d must <= %d(HARD_MAX_CONN_LIMIT)!" , _func_ , max_conn , HARD_MAX_CONN_LIMIT)
		return nil
	}

	//init
	init_redis_client(pclient, pconfig, redis_addr, auth, max_conn, normal_conn)

	//go routine to manager
	go pclient.redis_client_manage(pconfig, true)

	log.Info("%s success! addr:%s max_conn:%v normal:%v", _func_, redis_addr, max_conn, normal_conn)
	return pclient
}


/*Reset Addr
* @redis_addr:new redis addr; "" means no use
* @auth:new redis auth. "" means no use
* @max_conn:new max conn. <=0 means no use;(ATTENTION: max_conn only support expand,and <=HARD_MAX_CONN_LIMIT)
* @normal_conn: new normal conn. <=0 means no use
*/
func (pclient *RedisClient) Reset(redis_addr string , auth string , max_conn int , normal_conn int) {
	pclient.reset_attr.addr = redis_addr
	pclient.reset_attr.auth = auth
	pclient.reset_attr.max_count = max_conn
	pclient.reset_attr.normal_count = normal_conn
	pclient.reset_queue <- true
	return
}




//redis exe cmd Asynchronously
//warning:if cb_arg includes pointer , you may not change it's member unlese new memory.
func (pclient *RedisClient) RedisExeCmd(pconfig *CommConfig, cb_func RedisCallBack, cb_arg []interface{}, cmd string, arg ...interface{}) {
	//check blocked queue
	len_block := len(pclient.block_queue)
	if len_block >= cap(pclient.block_queue) {
		pconfig.Log.Err("RedisExeCmd failed! block routine too may! please check system! %d", len_block)
		return
	}

	//start a routine
	go func() {
		defer func() {
			if err := recover(); err != nil {
				pclient.comm_config.Log.Err("redis exe cmd panic! err:%v" , err)
				return
			}
		}()

		var _func_ = "<RedisExeCmd>"
		log := pconfig.Log

		//throw block
		pclient.block_queue <- 1
		//log.Debug("%s block:%d" , _func_ , len(pclient.block_queue));


		//occupy connection
		var idx int = -1
		for i:=0; i<5; i++ { //try best to occupy a valid connection
			idx = <-pclient.idle_queue
			//log.Debug("%s get idle idx:%d remain:%d " , _func_ , idx , len(pclient.idle_queue));

			//check idx valid(if reset may let it invalid!)
			if pclient.conn_stats[idx] != REDIS_CONN_DONE || pclient.conns[idx] == nil {
				log.Err("%s connection not valid! idx:%d", _func_, idx)
				continue
			}

			//valid
			break
		}
		if idx < 0 || pclient.conn_stats[idx]!=REDIS_CONN_DONE || pclient.conns[idx]==nil{
			log.Err("%s valid connection still not found! will drop request!", _func_)
			<-pclient.block_queue
			return
		}

		//exe cmd
		conn := pclient.conns[idx]
		reply, err := conn.Do(cmd, arg...)
		//free connection
		if idx < pclient.max_count { //valid idx will put again
			pclient.idle_queue <- idx
		}
		//unblock
		<-pclient.block_queue

		// handle result
		if err != nil {
			log.Err("%s failed! cmd:%v err:%v ", _func_, cmd, err)
			if cb_func != nil {
				cb_func(pconfig, err, cb_arg) //return err as result
			}
			return
		}

		//call-back
		if cb_func != nil {
			cb_func(pconfig, reply, cb_arg)
		}

	}()
}

//Alloc Synchronise Cmd Head
//Warning:
//1.This Method Must Be Put in an independent go-routine or will block main process
//2.Must Release Head After using head
func (pclient *RedisClient) AllocSyncCmdHead() *SyncCmdHead {
	var _func_ = "<redis_client.AllocSyncCmdHead>"
	pconfig := pclient.comm_config
	log := pconfig.Log

	//check blocked queue
	len_block := len(pclient.block_queue)
	if len_block >= cap(pclient.block_queue) {
		log.Err("RedisExeCmd failed! block routine too may! please check system! %d", len_block)
		return nil
	}

	//throw block
	pclient.block_queue <- 1


	//occupy connection
	var idx int = -1
	for i:=0; i<5; i++ { //try best to occupy a valid connection
		idx = <-pclient.idle_queue
		//log.Debug("%s get idle idx:%d remain:%d " , _func_ , idx , len(pclient.idle_queue));

		//check idx valid(if reset may let it invalid!)
		if pclient.conn_stats[idx] != REDIS_CONN_DONE || pclient.conns[idx] == nil {
			log.Err("%s connection not valid! idx:%d", _func_, idx)
			continue
		}

		//valid
		break
	}
	if idx < 0 || pclient.conn_stats[idx]!=REDIS_CONN_DONE || pclient.conns[idx]==nil{
		log.Err("%s valid connection still not found! will drop request!", _func_)
		<-pclient.block_queue
		return nil
	}

	//return success
	phead := new(SyncCmdHead)
	phead.idx = idx
	phead.conn = pclient.conns[idx]
	return phead
}

// redis exe cmd synchronised
//warning:synchronize cmd should be in an independent go-routine
func (pclient *RedisClient) RedisExeCmdSync(phead *SyncCmdHead , cmd string, arg ...interface{}) (interface{} , error){
	defer func() {
		if err := recover(); err != nil {
			pclient.comm_config.Log.Err("redis exe cmd panic! err:%v" , err)
			return
		}
	}()

	if phead == nil || phead.conn==nil {
		return nil , errors.New("phead nil!")
	}

	reply, err := phead.conn.Do(cmd, arg...)
	if err != nil {
		return nil , err
	}
	return reply , nil
}


//Release Synchronise Cmd Head
func (pclient *RedisClient) FreeSyncCmdHead(phead *SyncCmdHead) {
	//free connection
	if phead.idx < pclient.max_count { //valid idx will put again
		pclient.idle_queue <- phead.idx
	}
	//unblock
	<-pclient.block_queue
	pclient.comm_config.Log.Debug("redis_client.FreeSyncCmdHead Finish")
}


func (pclient *RedisClient) Close(pconfig *CommConfig) {
	//close_redis_conn(pclient, pconfig, nil)
	pclient.err_queue <- 1
}

func (pclient *RedisClient) GetConnNum() int {
	return pclient.conn_count
}

/*--------------------------Static Func----------------------------*/
//init redis_conn
func init_redis_client(pclient *RedisClient, pconfig *CommConfig, addr string, auth string, max_conn, normal_conn int) {
	var _func_ = "<init_redis_client>"
	log := pconfig.Log

	//set pconn info
	pclient.comm_config = pconfig
	pclient.addr = addr
	pclient.auth = auth
	pclient.max_count = max_conn
	pclient.normal_count = normal_conn

	//pre alloc uplimit
	pclient.conn_stats = make([]int8, HARD_MAX_CONN_LIMIT)
	pclient.conns = make([]redis.Conn, HARD_MAX_CONN_LIMIT)
	pclient.exit_queue = make(chan bool , 1)                      //exit
	pclient.idle_queue = make(chan int, HARD_MAX_CONN_LIMIT) //non-block
	pclient.err_queue = make(chan int, HARD_MAX_CONN_LIMIT)
	pclient.block_queue = make(chan int8, block_queue_len)
	pclient.reset_queue = make(chan bool , 10) //avoid block

	log.Info("%s success! addr:%s", _func_, addr)
	return
}

//connect most count connection to redis from pclient
func connect_redis_num(pclient *RedisClient, pconfig *CommConfig, count int) {
	var _func_ = "<connect_redis_num>"
	var num int = 0
	log := pconfig.Log

	//each go-routine to connect
	for i := 0; i < pclient.max_count; i++ {
		//check num
		if num >= count {
			break
		}

		//need non-connect
		if pclient.conn_stats[i] != REDIS_CONN_NONE {
			continue
		}

		//go connect
		pclient.conn_stats[i] = REDIS_CONN_ING
		num = num + 1

		//try connect each
		go func(idx int) {
			conn, err := redis.Dial("tcp", pclient.addr)
			if err != nil {
				log.Err("%s to %s failed! idx:%d err:%v", _func_, pclient.addr, idx, err)
				pclient.conn_stats[idx] = REDIS_CONN_NONE
				return
			}

			//auth
			if len(pclient.auth) > 0 {
				_, err = conn.Do("AUTH", pclient.auth)
				if err != nil {
					log.Err("%s auth failed! err:%s", _func_, err)
					conn.Close()
					pclient.conn_stats[idx] = REDIS_CONN_NONE
					return
				}
				//log.Info("%s auth to %s ids:%d success!" , _func_ , pclient.addr , idx);
			}

			//add count
			pclient.Lock()
			pclient.conn_count = pclient.conn_count + 1
			pclient.conn_stats[idx] = REDIS_CONN_DONE
			pclient.conns[idx] = conn
			pclient.Unlock()

			//idle
			pclient.idle_queue <- idx
			log.Info("%s to %s done! idx:%d conn:%d", _func_, pclient.addr, idx, pclient.conn_count)
		}(i)

	}

	log.Info("%s finish! num:%d count:%d", _func_, num, count)
}

//if spec_conn is nil means close all connection
func close_redis_conn(pclient *RedisClient, spec_conn []int) {
	var _func_ = "<close_redis_conn>"
	log := pclient.comm_config.Log

	//close all conn
	if spec_conn == nil {
		//try empty all idle info
		idle_num := len(pclient.idle_queue)
		for i:=0; i<idle_num; i++ {
			select {
			case <-pclient.idle_queue:
				//nothing
			default:
				//nothing
			}
		}

		pclient.Lock()
		//close all
		for i := 0; i < pclient.max_count; i++ {
			if pclient.conn_stats[i]!=REDIS_CONN_NONE && pclient.conns[i]!=nil {
				err := pclient.conns[i].Close()
				if err != nil {
					log.Err("%s close spec conn:%d failed! addr:%s err:%v", _func_, i, pclient.addr, err)
				} else {
					log.Info("%s close spec conn:%d success! addr:%s", _func_, i, pclient.addr)
				}
			}
			pclient.conn_stats[i] = REDIS_CONN_NONE
			pclient.conns[i] = nil
		}

		//reset
		pclient.conn_count = 0
		pclient.Unlock()
		return
	}

	//close spec_conn
	pclient.Lock()
	for i := 0; i < len(spec_conn); i++ {
		idx := spec_conn[i]
		log.Info("%s try-close spec conn:%d", _func_, idx)
		if pclient.conn_stats[idx]!=REDIS_CONN_NONE && pclient.conns[idx]!=nil {
			err := pclient.conns[idx].Close()
			if err != nil {
				log.Err("%s close spec conn:%d failed! addr:%s err:%v", _func_, idx, pclient.addr, err)
			} else {
				log.Info("%s close spec conn:%d success! addr:%s", _func_, idx, pclient.addr)
			}

			if pclient.conn_stats[idx] == REDIS_CONN_DONE {
				pclient.conn_count--
			}
		}

		pclient.conn_stats[idx] = REDIS_CONN_NONE
		pclient.conns[idx] = nil
	}
	pclient.Unlock()

}

//manage each connection of client
func (pclient *RedisClient) redis_client_manage(pconfig *CommConfig, start bool) {
	var _func_ = "<redis_client_manage>"
	log := pconfig.Log

	if start {
		connect_redis_num(pclient, pconfig, pclient.normal_count)
	}
	for {
		time.Sleep(check_conn_circle * time.Second)
		//check exit
		select {
		case _ = <-pclient.exit_queue:
			log.Info("%s detect exit flg!", _func_)
			//close all connection
			close_redis_conn(pclient , nil)
			return
		default: //nothing
			//nothing
		}

		//close err connection
		if len(pclient.err_queue) > 0 {
			err_count := len(pclient.err_queue)
			spec_conn := make([]int, err_count)
			var idx int = -1
			for i:=0; i<err_count; i++ {
				idx = <- pclient.err_queue
				spec_conn[i] = idx
			}
			log.Info("%s try to close err connection! err_conn:%v", _func_, spec_conn)
			close_redis_conn(pclient, spec_conn)
		}

		//check reset
		if len(pclient.reset_queue) > 0 {
			for i:=0; i<len(pclient.reset_queue); i++{
				select {
				case <- pclient.reset_queue: //empty queue
					//nothing
				default:
					//nothing
				}
			}
			pclient.reset_redis_attr()
		}


		//connect to normal
		curr_conn := pclient.conn_count
		num := pclient.normal_count - curr_conn
		if num > 0 {
			log.Info("%s will connect to normal! num:%d normal:%d curr:%d", _func_, num, pclient.normal_count, curr_conn)
			connect_redis_num(pclient, pconfig, num) //connect num
			continue
		}

		//expand connection if busy
		idle_num := len(pclient.idle_queue)
		block_num := len(pclient.block_queue)
		/*log.Debug("%s idle_num:%d block_num:%d curr:%d normal:%d max:%d" , _func_ , idle_num , block_num , curr_conn , pclient.normal_count ,
		  pclient.max_count);*/
		if idle_num <= 2 || block_num > 0 { //busy
			expand_count := int((pclient.max_count - curr_conn) / 3)
			log.Info("%s busy will expand connection! idle:%d block:%d curr_conn:%d max_conn:%d expand:%d", _func_, idle_num, block_num,
				curr_conn, pclient.max_count, expand_count)
			if expand_count < 0 {
				//nothing to do.
				log.Info("%s up limit connection!! curr:%d", _func_, curr_conn)
			} else {
				connect_redis_num(pclient, pconfig, expand_count)
			}
			continue
		}

		//shrink connection if idle
		if idle_num > pclient.normal_count { //free
			shrink_count := int((idle_num - pclient.normal_count) / 3)
			if shrink_count <= 0 {
				shrink_count = idle_num - pclient.normal_count
			}

			log.Info("%s free will release connection! idle:%d curr_conn:%d normal_conn:%d shrink:%d", _func_, idle_num, curr_conn, pclient.normal_count,
				shrink_count)
			release_list := make([]int, shrink_count)
			for i := 0; i < shrink_count; i++ {
				release_list[i] = <-pclient.idle_queue
			}

			close_redis_conn(pclient, release_list)
		}

	}
}

func (pclient *RedisClient) reset_redis_attr() {
	var _func_ = "<RedisClient.reset_redis_attr>"
	log := pclient.comm_config.Log

	//check reset attr
	if (pclient.reset_attr.addr!="" && pclient.reset_attr.addr!=pclient.addr) || (pclient.reset_attr.auth!="" && pclient.reset_attr.auth!=pclient.auth) {
		log.Info("%s addr or pass chged! addr:%s-->%s pass:%s-->%s" , _func_ , pclient.addr , pclient.reset_attr.addr ,
			pclient.auth , pclient.reset_attr.auth)

		//close all conn
		close_redis_conn(pclient , nil)

		//reset addr & pass
		if pclient.reset_attr.addr != "" {
			pclient.addr = pclient.reset_attr.addr
		}

		if pclient.reset_attr.auth != "" {
			pclient.auth = pclient.reset_attr.auth
		}

	}

	//check max-conn
	for {
		if pclient.reset_attr.max_count<=0 || pclient.max_count==pclient.reset_attr.max_count {
			//nothing
			break
		}

		if pclient.reset_attr.max_count > HARD_MAX_CONN_LIMIT {
			log.Err("%s failed! max_conn:%d should <= %d", _func_, pclient.reset_attr.max_count, HARD_MAX_CONN_LIMIT)
			break
		}

		if pclient.reset_attr.max_count < pclient.max_count {
			log.Info("%s warning! new max_count:%d < curr max_count:%d! it may result some request fail!" , _func_ ,
				pclient.reset_attr.max_count , pclient.max_count)
		}

		log.Info("%s max_count %d --> %d" , _func_ , pclient.max_count , pclient.reset_attr.max_count)
		pclient.max_count = pclient.reset_attr.max_count
		break
	}

	//check normal-conn
	if pclient.reset_attr.normal_count>0  && pclient.reset_attr.normal_count!=pclient.normal_count{
		if pclient.reset_attr.normal_count > pclient.max_count {
			log.Err("%s normal_count:%d > max_count:%d. set fail!" , _func_ , pclient.reset_attr.normal_count ,
				pclient.max_count)
		} else {
			log.Info("%s normal_count %d --> %d" , _func_ , pclient.normal_count , pclient.reset_attr.normal_count)
			pclient.normal_count = pclient.reset_attr.normal_count
		}
	}

    log.Info("%s finish! reset attr:%v" , _func_ , pclient.reset_attr)
	pclient.reset_attr.normal_count = 0
	pclient.reset_attr.max_count = 0
	pclient.reset_attr.addr = ""
	pclient.reset_attr.auth = ""
	return
}