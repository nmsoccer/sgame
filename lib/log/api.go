package log

import (
    "fmt"
    "time"
    "bytes"
)

const (
	TIME_FORMAT_SEC="2006-01-02 15:04:05"
	TIME_FORMAT_MILL="2006-01-02 15:04:05.000"
	TIME_FORMAT_MICR="2006-01-02 15:04:05.000000"
	TIME_FORMAT_NANO="2006-01-02 15:04:05.000000000"
	
	DEBUG = SL_DEBUG
	INFO = SL_INFO
	ERR = SL_ERR
	FATAL = SL_FATAL
	LOG_CH_SIZE=4098 //channel of log
)

type LogHeader interface {
	Log(log_level int , format string, arg ...interface{})
	Debug(format string , arg ...interface{})
	Info(format string , arg ...interface{})
	Err(format string , arg ...interface{})
	Close()
}

////////////////SPEC LOG STRUCT//////////////
type log_item struct {
	level int
	log_v string
}


type SLogPen struct {
	format string //refer TIME_FORMAT_xx
	handler int
	log_ch chan *log_item; 
}


//Open a Log handler
func OpenLog(log_name string) (*SLogPen) {
	lp := new(SLogPen);
	ret := SLogLocalOpen(SL_DEBUG, log_name, SLF_RAW , SLD_MILL , (50*1024*1024), 5);
	if ret < 0 {
		fmt.Printf("open log %s failed!", log_name);
		return nil;
	}
	
	lp.log_ch = make(chan *log_item , LOG_CH_SIZE);	
	lp.handler = ret;
	lp.format = TIME_FORMAT_MILL;
	//start writing goroutine
	go lp.write_log();	
	return lp;
}



func (lp *SLogPen) Log(log_level int , format string, arg ...interface{}) {
	lp.record_log(log_level, format, arg...);
	//SLog(lp.handler, log_level, format, arg...);
}

func (lp *SLogPen) Debug(format string , arg ...interface{}) {
	lp.record_log(DEBUG, format, arg...);
	//SLog(lp.handler , DEBUG , format , arg...);
}

func (lp *SLogPen) Info(format string , arg ...interface{}) {
	lp.record_log(INFO, format, arg...);
	//SLog(lp.handler , INFO , format , arg...);
}

func (lp *SLogPen) Err(format string , arg ...interface{}) {
	lp.record_log(ERR, format, arg...);
	//SLog(lp.handler , ERR , format , arg...);
}

func (lp *SLogPen) Close() {
	lp.record_log(-1, "%s" , "log closing...");
	//SLogClose(lp.handler);
}

/*-----------------------------STATIC FUNC----------------------*/
func (lp *SLogPen) record_log(log_level int , format string, arg ...interface{}) {	
	//convert str:
	prefix := lp.handle_prefix(log_level);
	content := fmt.Sprintf(format, arg...);
	var buffer bytes.Buffer;
	buffer.WriteString(prefix);
	buffer.WriteString(content);
	
	//append
	pitem := new(log_item);
	pitem.level = log_level;
	pitem.log_v = buffer.String();
	lp.log_ch <- pitem;
}

func (lp *SLogPen) write_log() {
	var pitem *log_item;
	for {
		pitem = <- lp.log_ch;
		if pitem.level >= 0 {
		    SLogBytes(lp.handler, pitem.level, []byte(pitem.log_v));
		} else { //exit
			SLogBytes(lp.handler, INFO , []byte(pitem.log_v));
			SLogClose(lp.handler);
			break;
		}
	}
}

// can't use slog prefix ,we should handle add
func (lp *SLogPen) handle_prefix(log_level int) string {
	label := "debug";
	switch log_level {
		case DEBUG:
		    label = "debug";
		case INFO:
		    label = "info";
		case ERR:
		    label = "err";
		case FATAL:
		    label = "fatal";
		default:
		    //nothing                
	}
	
	return fmt.Sprintf("[%s %s] ", time.Now().Format(lp.format) , label);
}

