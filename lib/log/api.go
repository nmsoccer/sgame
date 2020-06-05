package log

import (
    "fmt"
)

const (
	DEBUG = SL_DEBUG
	INFO = SL_INFO
	ERR = SL_ERR
	FATAL = SL_FATAL
)

type LogHeader interface {
	Log(log_level int , format string, arg ...interface{})
	Debug(format string , arg ...interface{})
	Info(format string , arg ...interface{})
	Err(format string , arg ...interface{})
	Close()
}

////////////////SPEC LOG STRUCT//////////////
type SLogPen struct {
	handler int 
}

func OpenLog(log_name string) (*SLogPen) {
	lp := new(SLogPen);
	ret := SLogLocalOpen(SL_DEBUG, log_name, SLF_PREFIX , SLD_MILL , (50*1024*1024), 5);
	if ret < 0 {
		fmt.Printf("open log %s failed!", log_name);
		return nil;
	}
	
	lp.handler = ret;
	return lp;
}

func (lp *SLogPen) Log(log_level int , format string, arg ...interface{}) {
	SLog(lp.handler, log_level, format, arg...);
}

func (lp *SLogPen) Debug(format string , arg ...interface{}) {
	SLog(lp.handler , DEBUG , format , arg...);
}

func (lp *SLogPen) Info(format string , arg ...interface{}) {
	SLog(lp.handler , INFO , format , arg...);
}

func (lp *SLogPen) Err(format string , arg ...interface{}) {
	SLog(lp.handler , ERR , format , arg...);
}

func (lp *SLogPen) Close() {
	SLogClose(lp.handler);
}

