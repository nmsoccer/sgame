package log

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

const (
	//Attr
	LOG_ATTR_LEVEL = 1 //chg log filt level. refer LOG_LV_XX
	LOG_ATTR_DEGREEE = 2 //chg log time degree. refer LOG_DEGREE_XX
	LOG_ATTR_ROTATE = 3  //chg log rotate number
	LOG_ATTR_SIZE = 4    //chg log size must>1024

	//LOG-DEGREE
	LOG_DEGREE_SEC = 1  //sec
	LOG_DEGREE_MIL = 2  //milli sec
	LOG_DEGREE_MIC = 3  //micro sec
	LOG_DEGREE_NANO = 4 //nano sec


	//TIME-FORMAT
	TIME_FORMAT_SEC  = "2006-01-02 15:04:05"
	TIME_FORMAT_MILL = "2006-01-02 15:04:05.000"
	TIME_FORMAT_MICR = "2006-01-02 15:04:05.000000"
	TIME_FORMAT_NANO = "2006-01-02 15:04:05.000000000"

	//LOG-LEVEL
	LOG_LV_DEBUG       = SL_DEBUG //debug
	LOG_LV_INFO        = SL_INFO  //info
	LOG_LV_ERR         = SL_ERR  //err
	LOG_LV_FATAL       = SL_FATAL  //fatal
	LOG_CH_SIZE = 4098 //channel of log


	//DEFAULT
	LOG_DEFAULT_FILT_LEVEL = LOG_LV_DEBUG
	LOG_DEFAULT_DEGREE = LOG_DEGREE_MIL
	LOG_DEFAULT_ROTATE = 5 //default rotate 5
	LOG_DEFAULT_SIZE = (50*1024*1024) //default size:50M

)

type LogHeader interface {
	Log(log_level int, format string, arg ...interface{})
	Debug(format string, arg ...interface{})
	Info(format string, arg ...interface{})
	Err(format string, arg ...interface{})
	Fatal(format string, arg ...interface{})
	ChgAttr(attr int , value int) int
	Close()
}

////////////////SPEC LOG STRUCT//////////////
type log_item struct {
	level int
	//log_v string
	log_v []byte
}

type SLogPen struct {
	filt_level int //refer LOG_LV_XX
	log_degree int //refer LOG_DEGREE_XX
	handler int
	log_ch  chan *log_item
}

////////////GLOBAL VAR///////////////////////
var queue_pool sync.Pool; //store log_item
var log_degree_fmt []string //log_degree --> time-format
var log_level_label []string //log_level --> label

func init() {
	//log_degree_fmt
	log_degree_fmt = make([]string , LOG_DEGREE_NANO+1)
	log_degree_fmt[0] = TIME_FORMAT_SEC //default
	log_degree_fmt[LOG_DEGREE_SEC] = TIME_FORMAT_SEC
	log_degree_fmt[LOG_DEGREE_MIL] = TIME_FORMAT_MILL
	log_degree_fmt[LOG_DEGREE_MIC] = TIME_FORMAT_MICR
	log_degree_fmt[LOG_DEGREE_NANO] = TIME_FORMAT_NANO

	//log_level_label
	log_level_label = make([]string , LOG_LV_FATAL+1)
	log_level_label[0] = "debug" //default
	log_level_label[LOG_LV_DEBUG] = "debug"
	log_level_label[LOG_LV_INFO] = "info"
	log_level_label[LOG_LV_ERR] = "err"
	log_level_label[LOG_LV_FATAL] = "fatal"

	//new
	queue_pool.New = func() interface{} {return new(log_item)}
}


/*Open a Log handler
* @filt_level:refer LOG_LV_XX. only print log above filt-level
* @log_degree:refer LOG_DEGREE_XX
* @rotate: log file max rotate num
* @size: log file size
*/
func OpenLog(log_name string , filt_level int , log_degree int , rotate int , log_size int) *SLogPen {
	lp := new(SLogPen)
	//set default
	if filt_level<LOG_LV_DEBUG || filt_level>LOG_LV_FATAL {
		filt_level = LOG_DEFAULT_FILT_LEVEL
	}
	if log_degree<LOG_DEGREE_SEC || filt_level>LOG_DEGREE_NANO {
		log_degree = LOG_DEFAULT_DEGREE
	}
	if rotate <= 0 {
		rotate = LOG_DEFAULT_ROTATE
	}
	if log_size <= 0 {
		log_size = LOG_DEFAULT_SIZE
	}

	//OPEN
	ret := SLogLocalOpen(filt_level, log_name, SLF_RAW, SLD_SEC, log_size, rotate)
	if ret < 0 {
		fmt.Printf("open log %s failed!", log_name)
		return nil
	}

	lp.log_ch = make(chan *log_item, LOG_CH_SIZE)
	lp.handler = ret
	lp.log_degree = log_degree
	//lp.time_format = TIME_FORMAT_MILL
	lp.filt_level = filt_level

	//start writing goroutine
	go lp.write_log()
	return lp
}

func (lp *SLogPen) Log(log_level int, format string, arg ...interface{}) {
	lp.record_log(log_level, format, arg...)
	//SLog(lp.handler, log_level, format, arg...);
}

func (lp *SLogPen) Debug(format string, arg ...interface{}) {
	lp.record_log(LOG_LV_DEBUG, format, arg...)
	//SLog(lp.handler , DEBUG , format , arg...);
}

func (lp *SLogPen) Info(format string, arg ...interface{}) {
	lp.record_log(LOG_LV_INFO, format, arg...)
}

func (lp *SLogPen) Err(format string, arg ...interface{}) {
	lp.record_log(LOG_LV_ERR, format, arg...)
}

func (lp *SLogPen) Fatal(format string, arg ...interface{}) {
	lp.record_log(LOG_LV_FATAL, format, arg...)
}

func (lp *SLogPen) Close() {
	lp.record_log(-1, "%s", "log closing...")
	//SLogClose(lp.handler);
}

func (lp *SLogPen) ChgAttr(attr int , value int) int {
	var result int = -1;

	switch attr {
	case LOG_ATTR_LEVEL:
		if value<LOG_LV_DEBUG || value>LOG_LV_FATAL {
			break;
		}
		lp.filt_level = value //may critical race
		result = 0
	case LOG_ATTR_DEGREEE:
		if value<LOG_DEGREE_SEC || value>LOG_DEGREE_NANO {
			break
		}
		lp.log_degree = value
		result = 0
	case LOG_ATTR_ROTATE:
		if value < 0 {
			break
		}
		result = SLogChgAttr(lp.handler, -1 , -1, -1, value, -1);
	case LOG_ATTR_SIZE:
		if value <= 0 {
			break
		}
		result = SLogChgAttr(lp.handler, -1 , -1, value, -1, -1);
	default:
		break
	}
	return result
}



/*-----------------------------STATIC FUNC----------------------*/
func (lp *SLogPen) record_log(log_level int, format string, arg ...interface{}) {
	//filt
	if log_level < lp.filt_level {
		return;
	}

	//convert str:
	prefix := lp.handle_prefix(log_level)
	content := fmt.Sprintf(format, arg...)
	var buffer bytes.Buffer
	buffer.WriteString(prefix)
	buffer.WriteString(content)

	//get item
	//pitem := new(log_item)
	var pitem *log_item
	pv := queue_pool.Get()
	pitem , ok := pv.(*log_item)
	if !ok {
		pitem = new(log_item)
	}

	//store
	log_v := buffer.Bytes()
	pitem.level = log_level
	pitem.log_v = log_v
	lp.log_ch <- pitem
}

func (lp *SLogPen) write_log() {
	var pitem *log_item
	for {
		pitem = <-lp.log_ch
		if pitem.level >= 0 {
			SLogBytes(lp.handler, pitem.level, pitem.log_v)
			queue_pool.Put(pitem)
		} else { //exit
			SLogBytes(lp.handler, LOG_LV_INFO, pitem.log_v)
			SLogClose(lp.handler)
			break
		}
	}
}

// can't use slog prefix ,we should handle add
func (lp *SLogPen) handle_prefix(log_level int) string {
	if log_level<0 || log_level>LOG_LV_FATAL {
		log_level = 0
	}
	log_degree := lp.log_degree
	if log_degree<0 || log_degree>LOG_DEGREE_NANO {
		log_degree = 0
	}

	return fmt.Sprintf("[%s %s] ", time.Now().Format(log_degree_fmt[log_degree]), log_level_label[log_level])
}
