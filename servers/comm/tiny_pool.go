package comm

import (
	"math/rand"
	"time"
)

/*
TINY POOL STORE SIMPLE DATA
MAY REFER sync.Pool
 */

type TinyNewFunc func() interface{}

type TinyPool struct{
	pool_size int
	data chan interface{}
	new_func TinyNewFunc
}

//New TinyPool
func NewTinyPool(pool_size int , new_func TinyNewFunc) *TinyPool {
	pool := new(TinyPool);
	pool.pool_size = pool_size;
	pool.data = make(chan interface{} , pool_size);
	pool.new_func = new_func;
	go pool.serve();
	return pool;
}

//del mem
func (pool *TinyPool) serve() {
	for {
		time.Sleep(1 * time.Second);
		num := len(pool.data);
		if num <=0 {
			continue;
		}
		del_count := rand.Intn(num);
		for i := 0; i < del_count; i++ {
			<-pool.data;
		}
	}
}


//Get an obj
func (pool *TinyPool) Get() interface{} {
	select {
	case pv := <- pool.data:
		return pv;
	default:
		if pool.new_func != nil {
			return pool.new_func();
		} else {
			return nil;
		}
	}
}

//Put an obj
func (pool *TinyPool) Put(v interface{}) {
	if len(pool.data) >= cap(pool.data) {
		return;
	}

	pool.data <- v;
}

