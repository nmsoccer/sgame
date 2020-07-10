package comm

import (
	"time"
)

const (
	TICKER_TYPE_SINGLE int8 = 1
	TICKER_TYPE_CIRCLE int8 = 2
	MAX_EXPIRE_MS int64 = 0xFFFFFFFFFFFF

	PERIOD_HEART_BEAT_DEFAULT=10000 //10s
	PERIOD_REPORT_SYNC_DEFAULT=60000 //1min

)

type TickFunc func(arg interface{});


type ticker struct {
	ctype int8 //refer TICKER_TYPE_SINGLE
	name string
    expire_ms int64 //expire time (ms)
	period_ms int64 //period ms
	callback TickFunc
	arg interface{}
	shot int64 //shot count
	next *ticker
}

type TickPool struct {
	config *CommConfig
	head ticker
	count int
	latest_expire_ms int64
}

//create tick pool
func NewTickPool(pconfig *CommConfig) *TickPool {
    var _func_ = "<NewTickPool>";
    log := pconfig.Log;

    //new
    ppool := new(TickPool);
    if ppool == nil {
    	log.Err("%s faile! new pool failed!" , _func_);
    	return nil;
	}

	ppool.config = pconfig;
	ppool.count = 0;
	ppool.head.name = "head";
	ppool.head.next = nil;
	ppool.latest_expire_ms = MAX_EXPIRE_MS;
	return ppool;
}

/* add ticker
@ctype:ticker type refer TICKER_TYPE_XX
@start_ms:started ms. <=0:started from current >0 futrue timestamp
@period_ms:interval timeing
@callback:trig function if not nil
@arg: arg for callback func if not nil
*/
func (pool *TickPool) AddTicker(name string , ctype int8 , start_ms int64 , period_ms int64 , callback TickFunc , arg interface{}) bool {
    var _func_ = "<AddTicker>";
	log := pool.config.Log;

    //check type
    if ctype!=TICKER_TYPE_SINGLE && ctype!=TICKER_TYPE_CIRCLE {
    	log.Err("%s failed! tick type:%d illegal!" , _func_ , ctype);
    	return false;
	}

	if period_ms < 0 {
		log.Err("%s failed! period:%d illegal!" , _func_ , period_ms);
		return false;
	}

    //new
    pticker := new(ticker);
    if pticker == nil {
    	log.Err("%s failed! new ticker fail!" , _func_);
    	return false;
	}
	curr_ms := time.Now().UnixNano()/1000/1000;

	//set ticker
	pticker.ctype = ctype;
	pticker.name = name;
	if start_ms <= 0 {
		pticker.expire_ms = curr_ms + period_ms;
	} else {
		pticker.expire_ms = start_ms;
	}
    pticker.period_ms = period_ms;
    pticker.callback = callback;
    pticker.arg = arg;

    //append to pool
    pticker.next = pool.head.next;
    pool.head.next = pticker;
    pool.count += 1;
    if pticker.expire_ms < pool.latest_expire_ms {
    	pool.latest_expire_ms = pticker.expire_ms;
	}

    log.Info("%s success! name:%s type:%d period:%v expire:%v ticker_count:%d latest_expire:%v" , _func_ , name , ctype , period_ms ,
    	pticker.expire_ms , pool.count , pool.latest_expire_ms);
    return true;
}

/*tick
*@sleep_us if>0 sleep xx microseconds else no sleep
*/
func (pool *TickPool) Tick(sleep_us int64) {
	var _func_ = "TickPool.Tick";
	log := pool.config.Log;

	if sleep_us > 0 {
		time.Sleep(time.Duration(sleep_us) * time.Microsecond);
	}
	//count
	if pool.count<= 0 {
		return;
	}

	//check ms
	curr_ms := time.Now().UnixNano()/1000/1000;
	var new_latest_ms int64 = MAX_EXPIRE_MS;
    if curr_ms < pool.latest_expire_ms {
    	return;
	}

	//check ticker
    var ptick *ticker = pool.head.next;
    var pprev *ticker = &pool.head;

    for {
        if ptick == nil {
        	break;
		}

		//no expire
        if ptick.expire_ms > curr_ms {
        	if ptick.expire_ms < new_latest_ms {
        		new_latest_ms = ptick.expire_ms;
			}
			//log.Debug("%s no expire! name:%s expire_ts:%v new_latest_ts:%v curr:%v" , _func_ , ptick.name ,
			//	ptick.expire_ms , new_latest_ms , curr_ms);
			pprev = ptick;
			ptick = ptick.next;
        	continue;
		}

		//expired
		//exe callback
		//log.Debug("%s trig name:%s period:%v expire_ms:%v now:%v type:%d " , _func_ , ptick.name , ptick.period_ms , ptick.expire_ms ,
		//	curr_ms , ptick.ctype);
		if ptick.callback != nil {
			ptick.callback(ptick.arg);
		}

		//single shot
		if ptick.ctype == TICKER_TYPE_SINGLE { //remove this ticker
			log.Info("%s remove single-shot ticker:%s tick_count:%d" , _func_ , ptick.name , pool.count-1);
			pprev.next = ptick.next;
			ptick = ptick.next;
			pool.count -= 1;
			continue;
		}

		//circle
		ptick.expire_ms = curr_ms + ptick.period_ms;
        if ptick.expire_ms < new_latest_ms {
        	new_latest_ms = ptick.expire_ms;
		}
		ptick.shot += 1;
		//log.Debug("%s circle ticker:%s shot:%v next:%v" , _func_ , ptick.name , ptick.shot , ptick.expire_ms);

		//continue
        pprev = ptick;
        ptick = ptick.next;
	}

	//reset latest
	//log.Debug("%s reset expire %v --> %v" , _func_ , pool.latest_expire_ms , new_latest_ms);
    pool.latest_expire_ms = new_latest_ms;
}