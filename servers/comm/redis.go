package comm

import (
	"github.com/gomodule/redigo/redis"
	"time"
	"sync"
)

const (
  REDIS_CONN_NONE = iota
  REDIS_CONN_ING
  REDIS_CONN_DONE
)

var last_check int64;
const (
    check_conn_circle = 5;
    block_queue_len = (100000); //max block go-routine ...
)

type RedisClient struct {
	sync.Mutex
	addr string
	auth string
	max_count int
	normal_count int
	conn_count int	
	conn_stats []int8 //each conn stat refer REDIS_CONN_XX
	conns []redis.Conn  //each conn
	exit_queue chan bool //exit flag
	idle_queue chan int // idle index of conn
	err_queue chan int //err connection done-->err
	block_queue chan int8 //block request. cap is unlimited. init:1M
}

//call back
type RedisCallBack func(pconfig *CommConfig , result interface{} , cb_arg []interface {});

//Conver Wrapper refer reply.go in goredis
func Conv2Int(result interface{}) (int , error) {
	return redis.Int(result, nil);
}
func Conv2Int64(result interface{}) (int64 , error) {
	return redis.Int64(result, nil);
}
func Conv2UInt64(result interface{}) (uint64 , error) {
	return redis.Uint64(result, nil);
}
func Conv2Float64(result interface{}) (float64 , error) {
	return redis.Float64(result, nil);
}
func Conv2String(result interface{}) (string , error) {
	return redis.String(result, nil);
}
func Conv2Bytes(result interface{}) ([]byte , error) {
	return redis.Bytes(result, nil);
}
func Conv2Values(result interface{}) ([]interface{} , error) {
	return redis.Values(result, nil);
}
func Conv2Strings(result interface{}) ([]string , error) {
	return redis.Strings(result, nil);
}
func Conv2StringMap(result interface {}) (map[string]string , error) {
	return redis.StringMap(result, nil);
}
func Conv2IntMap(result interface {}) (map[string]int , error) {
	return redis.IntMap(result, nil);
}
func Conv2Int64Map(result interface {}) (map[string]int64 , error) {
	return redis.Int64Map(result, nil);
}



//New RedisClient
func NewRedisClient(pconfig *CommConfig , redis_addr string , auth string , max_conn,normal_conn int) *RedisClient{
    var _func_ = "<NewRedisClient>";
    log := pconfig.Log;

    //new
    pclient := new(RedisClient);
    if pclient == nil {
    	log.Err("%s Failed! new fail!" , _func_);
    	return nil;
    }
    
    //init
    init_redis_client(pclient, pconfig, redis_addr, auth, max_conn, normal_conn);
        
    //go routine to manager
    go pclient.redis_client_manage(pconfig , true);
    
    log.Info("%s success! addr:%s max_conn:%v normal:%v" , _func_ , redis_addr , max_conn , normal_conn);
    return pclient;	
}


//redis exe cmd
func (pclient *RedisClient) RedisExeCmd(pconfig *CommConfig , cb_func RedisCallBack , cb_arg []interface{} , cmd string , arg ...interface{}) {
	//check blocked queue
	len_block := len(pclient.block_queue);
	if  len_block >= cap(pclient.block_queue) {
		pconfig.Log.Err("RedisExeCmd failed! block routine too may! please check system! %d" , len_block);
		return;
	}
	
	
	//start a routine
	go func() {
		var _func_ = "<RedisExeCmd>";
		log := pconfig.Log;		
		
		//throw block
		pclient.block_queue <- 1;
		//log.Debug("%s block:%d" , _func_ , len(pclient.block_queue));
		//occupy connection
		idx := <- pclient.idle_queue;
		//log.Debug("%s get idle idx:%d remain:%d " , _func_ , idx , len(pclient.idle_queue));
				
		//exe cmd
		conn := pclient.conns[idx];
		reply , err := conn.Do(cmd , arg...);
		  //free connection
		pclient.idle_queue <- idx
		  //unblock
		<- pclient.block_queue;
		
		  // handle result
		if err != nil {
			log.Err("%s failed! cmd:%v err:%v " , _func_ , cmd , err);
			return;
		}
				
		//call-back
		if cb_func != nil {
			cb_func(pconfig , reply , cb_arg);
		}
					
	}();
}

func (pclient *RedisClient) Close(pconfig *CommConfig) {
	close_redis_conn(pclient, pconfig, nil);
}

func (pclient *RedisClient) GetConnNum() int {
	return pclient.conn_count;
}


/*--------------------------Static Func----------------------------*/
//init redis_conn
func init_redis_client(pclient *RedisClient , pconfig *CommConfig , addr string , auth string , max_conn,normal_conn int) {
	var _func_ = "<init_redis_client>";
	log := pconfig.Log;
	
	//set pconn info
    pclient.addr = addr;
    pclient.auth = auth;
    pclient.max_count = max_conn;
    pclient.normal_count = normal_conn;
    pclient.conn_stats = make([]int8 , pclient.max_count);
    pclient.conns = make([]redis.Conn , pclient.max_count);
    pclient.exit_queue = make(chan bool); //exit
    pclient.idle_queue = make(chan int , pclient.max_count + 10); //non-block
    pclient.err_queue = make(chan int , pclient.max_count + 10);
    pclient.block_queue = make(chan int8 , block_queue_len);
    
    log.Info("%s success! addr:%s" , _func_ , addr);
	return;
}

//connect most count connection to redis from pclient
func connect_redis_num(pclient *RedisClient , pconfig *CommConfig , count int) {
	var _func_ = "<connect_redis_num>";
	var num int = 0;
	log := pconfig.Log;
	
	//each go-routine to connect
	for i:=0; i<pclient.max_count; i++ {
		//check num
		if num >= count {
			break;
		}
				
		//need non-connect 
		if pclient.conn_stats[i] != REDIS_CONN_NONE {
			continue;
		}
						
		//go connect
		pclient.conn_stats[i] = REDIS_CONN_ING;
		num = num + 1;
		
		//try connect each	
		go func(idx int) {
		    conn , err := redis.Dial("tcp", pclient.addr);
		    if err != nil {
		        log.Err("%s to %s failed! idx:%d err:%v" , _func_ , pclient.addr , idx , err);
		        pclient.conn_stats[idx] = REDIS_CONN_NONE;
		        return;    	
		    }
			
			//auth
			if len(pclient.auth) > 0 {
			    _ , err = conn.Do("AUTH" , pclient.auth);
	            if err != nil {
		            log.Err("%s auth failed! err:%s", _func_ , err);
		            conn.Close();
		            pclient.conn_stats[idx] = REDIS_CONN_NONE;
		            return;
	            }
                //log.Info("%s auth to %s ids:%d success!" , _func_ , pclient.addr , idx);
			}
			
			//add count
		    pclient.Lock();
		    pclient.conn_count  = pclient.conn_count + 1;
		    pclient.conn_stats[idx] = REDIS_CONN_DONE;
		    pclient.conns[idx] = conn;
		    pclient.Unlock();
			    
		    //idle		    
		    pclient.idle_queue <- idx;
		    log.Info("%s to %s done! idx:%d conn:%d" , _func_ , pclient.addr , idx , pclient.conn_count);		    		    	
		}(i);
		
	}
	
	log.Info("%s finish! num:%d count:%d" , _func_ , num , count);
}

//if spec_conn is nil means close all connection
func close_redis_conn(pclient *RedisClient , pconfig *CommConfig , spec_conn []int) {
	var _func_ = "<close_redis_conn>";
	log := pconfig.Log;
	
	//close all conn
	if spec_conn == nil {
		pclient.Lock();
		pclient.exit_queue <- true;
	    for i:=0; i<pclient.max_count; i++ {
		    if pclient.conn_stats[i] != REDIS_CONN_NONE {
			    err := pclient.conns[i].Close();
			    if err != nil {
				    log.Err("%s close spec conn:%d failed! addr:%s err:%v" , _func_ , i , pclient.addr , err);
			    } else {
				    log.Info("%s close spec conn:%d success! addr:%s" , _func_ , i , pclient.addr);
			    }
		    }
		    pclient.conn_stats[i] = REDIS_CONN_NONE;
		    pclient.conns[i] = nil;
	    }
	
	    //reset
	    pclient.conn_count = 0;
	    pclient.Unlock();
	    return;
	}
	
	//close spec_conn
	pclient.Lock();
	for i:=0; i<len(spec_conn); i++ {
		idx := spec_conn[i];
		log.Info("%s try-close spec conn:%d" , _func_ , idx);
		if pclient.conn_stats[idx] != REDIS_CONN_NONE {
			err := pclient.conns[idx].Close();
			if err != nil {
				log.Err("%s close spec conn:%d failed! addr:%s err:%v" , _func_ , idx , pclient.addr , err);
			} else {
				log.Info("%s close spec conn:%d success! addr:%s" , _func_ , idx , pclient.addr);
			}
			
			if pclient.conn_stats[idx] == REDIS_CONN_DONE {
			    pclient.conn_count--;
			}
		}
	
		pclient.conn_stats[idx] = REDIS_CONN_NONE;
		pclient.conns[idx] = nil;
	}
	pclient.Unlock();
		
}

//manage each connection of client
func (pclient *RedisClient) redis_client_manage(pconfig *CommConfig , start bool) {
	var _func_ = "<redis_client_manage>";
	log := pconfig.Log;
	
	if start {
		connect_redis_num(pclient, pconfig, pclient.normal_count);
	}
	for {
		time.Sleep(check_conn_circle * time.Second);	
		//check exit
		select {
			case _ = <- pclient.exit_queue:
			  log.Info("%s detect exit flg!" , _func_);
			  return;
			default: //nothing
			  //nothing
		}    
	        
	    //close err connection
	    if len(pclient.err_queue) > 0 {
	    	spec_conn := make([]int , pclient.max_count);
	    	spec_conn = spec_conn[:0];
	    	for idx := range pclient.err_queue {
	    	    spec_conn = append(spec_conn , idx);	
	    	}
	    	log.Info("%s try to close err connection! err_conn:%v" , _func_ , spec_conn);
	    	close_redis_conn(pclient, pconfig, spec_conn);
	    }
	    
	    //connect to normal
	    //pclient.Lock();
	    curr_conn := pclient.conn_count;
	    //pclient.Unlock();
	    num := pclient.normal_count - curr_conn;
	    if num > 0 {
	    	log.Info("%s will connect to normal! num:%d normal:%d curr:%d" , _func_ , num , pclient.normal_count , curr_conn);
	    	connect_redis_num(pclient, pconfig, num); //connect num
	    	continue;
	    } 
	    
	    //expand connection if busy
	    idle_num := len(pclient.idle_queue);
	    block_num := len(pclient.block_queue);
	    /*log.Debug("%s idle_num:%d block_num:%d curr:%d normal:%d max:%d" , _func_ , idle_num , block_num , curr_conn , pclient.normal_count , 
	        pclient.max_count);*/
	    if idle_num<=2 || block_num >  0 { //busy  		    	
	    	expand_count := int((pclient.max_count - curr_conn) / 3);
	    	log.Info("%s busy will expand connection! idle:%d block:%d curr_conn:%d max_conn:%d expand:%d" , _func_ , idle_num , block_num,
	    		curr_conn , pclient.max_count ,  expand_count);
	    	if expand_count < 0 {
	    		//nothing to do.
	    		log.Info("%s up limit connection!! curr:%d" , _func_ , curr_conn);
	    	} else {
	    		connect_redis_num(pclient, pconfig, expand_count);
	    	}
	    	continue;
	    }
    
        //shrink connection if idle
        if idle_num > pclient.normal_count { //free
        	shrink_count := int((idle_num - pclient.normal_count) / 3);
        	if shrink_count <= 0 {
        		shrink_count = idle_num - pclient.normal_count;
        	}
        	
        	log.Info("%s free will release connection! idle:%d curr_conn:%d normal_conn:%d shrink:%d" , _func_ , idle_num , curr_conn , pclient.normal_count ,  
        		shrink_count);
        	release_list := make([]int , shrink_count);
        	for i:=0; i<shrink_count; i++ {
        		release_list[i] = <- pclient.idle_queue;
        	}
        	
        	close_redis_conn(pclient, pconfig, release_list);
        }
        
	}	
}
