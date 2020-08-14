package log

import (
	"fmt"
	"unsafe"
)

/*
// This is a wrapper of slog APIs for golang
// slog lib and header should be installed first.

#cgo CFLAGS: -g
#cgo LDFLAGS: -lm -lslog
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <slog/slog.h>

static int wrap_slog_log(int sld , int log_level , char *log) {
	return slog_log(sld , log_level , "%s" , log);
}
*/
import "C"

//SLOG-TYPE
const (
	SLT_LOCAL = C.SLT_LOCAL //local file-log
	SLT_NET   = C.SLT_NET   //remote net-log
)

//SLOG-LEVEL
const (
	SL_VERBOSE = iota //C.SL_VERBOSE
	SL_DEBUG
	SL_INFO
	SL_ERR
	SL_FATAL
)

//SLOG-DEGREE
const (
	SLD_SEC  = iota //C.SLD_SEC second
	SLD_MILL        //milli sec
	SLD_MIC         //micro sec
	SLD_NANO        //nano sec
)

//SLOG-FORMAT
const (
	SLF_PREFIX = C.SLF_PREFIX //each log has prefix like [time level]
	SLF_RAW    = C.SLF_RAW    //raw info
)

/************API*****************/
/***
Open A Local SLOG Descriptor
@filt_level:Log filter.Only Print Log if LOG_LEVEL >= filt_level.
@log_name:log file (include path)
@format:format of log. if 0 then default is SLF_PREFIX,if sets to SLF_RAW,then print raw info.
@log_degree:refer SLD_xx.the timing degree of log. if 0 then default by seconds.
@log_size:max single log_file size.if 0 then sets to defaut 20M
@rotate:log file rotate limit.if 0 then sets to default 5
*RETVALUE:
*-1: FAILED
*>=0:ALLOCATED SLD(SLOG Descriptor)
*/
func SLogLocalOpen(filt_level int, log_name string, format int, log_degree int, log_size int, rotate int) int {
	var _func_ string = "SLogLocalOpen"
	//Arg Check
	if filt_level < SL_VERBOSE || filt_level > SL_FATAL {
		fmt.Printf("%s failed! filt_level illegal! please refer SL_XX\n", _func_)
		return -1
	}

	if log_name == "" {
		fmt.Printf("%s failed! log_name nil\n", _func_)
		return -1
	}

	//default
	if format < SLF_PREFIX || format > SLF_RAW {
		format = SLF_PREFIX
	}

	if log_degree < SLD_SEC || log_degree > SLD_NANO {
		log_degree = SLD_SEC
	}

	if log_size <= 0 {
		log_size = (20 * 1024 * 1024)
	}

	if rotate <= 0 {
		rotate = 5
	}

	//fulfill option
	var c_option C.SLOG_OPTION
	/*
		cs := C.CString(log_name);
		defer C.free(unsafe.Pointer(cs));
		C.memcpy(unsafe.Pointer(p) , unsafe.Pointer(cs) , len(log_name));
	*/
	//&c_option.type_value type:*[256]uint8 --> [256]byte -->byte[:]
	copy((c_option.type_value)[:], []byte(log_name))
	c_option.log_degree = C.SLOG_DEGREE(log_degree)
	c_option.format = C.SLOG_FORMAT(format)
	c_option.log_size = C.int(log_size)
	c_option.rotate = C.int(rotate)

	//fmt.Printf("local coption:%+v %T\n", c_option , &c_option);

	//open
	slogd := C.slog_open(C.SLT_LOCAL, C.SLOG_LEVEL(filt_level), &c_option, nil)
	return int(slogd)
}

/***
Open A Net SLOG Descriptor(Peer should listen on a UDP Port)
@filt_level:Log filter.Only Print Log if LOG_LEVEL >= filt_level.
@log_name:log file (include path)
@format:format of log. if 0 then default is SLF_PREFIX,if sets to SLF_RAW,then print raw info.
@log_degree:refer SLD_xx.the timing degree of log. if 0 then default by seconds.
@log_size:max single log_file size.if 0 then sets to defaut 20M
@rotate:log file rotate limit.if 0 then sets to default 5
*RETVALUE:
*-1: FAILED
*>=0:ALLOCATED SLD(SLOG Descriptor)
*/
func SLogNetOpen(filt_level int, ip string, port int, format int, log_degree int) int {
	var _func_ string = "SLogNetOpen"
	//Arg Check
	if filt_level < SL_VERBOSE || filt_level > SL_FATAL {
		fmt.Printf("%s failed! filt_level illegal! please refer SL_XX\n", _func_)
		return -1
	}

	if ip == "" || port <= 0 {
		fmt.Printf("%s failed! ip:port<%s:%d> illegal\n", _func_ , ip , port)
		return -1
	}

	//default
	if format < SLF_PREFIX || format > SLF_RAW {
		format = SLF_PREFIX
	}

	if log_degree < SLD_SEC || log_degree > SLD_NANO {
		log_degree = SLD_SEC
	}

	//fulfill option
	var c_option C.SLOG_OPTION
	//&c_option.type_value type:*[256]uint8
	sip := c_option.type_value[:64]
	copy(sip, []byte(ip)) //ip

	sport := c_option.type_value[64:]
	p_port := unsafe.Pointer(&port)
	copy(sport, ((*[4]byte)(p_port))[:]) //port

	c_option.log_degree = C.SLOG_DEGREE(log_degree)
	c_option.format = C.SLOG_FORMAT(format)

	//fmt.Printf("net coption:%+v %T\n", c_option , &c_option);

	//open
	slogd := C.slog_open(SLT_NET, C.SLOG_LEVEL(filt_level), &c_option, nil)
	return int(slogd)
}

/***
Close A SLOG Descriptor
*RETVALUE:
*-1: FAILED
*=0: SUCCESS
*/
func SLogClose(sld int) int {
	result := C.slog_close(C.int(sld))
	return int(result)
}

/***
Log
@sld:opened slog descriptor
@log_level:refer to SL_XX.the level of this log.If log_level < filt_level(slog_open),it will not printed.
@RETVALUE:
-1:failed(err msg can be found in slog.log). 0:success
*/
func SLog(sld int, log_level int, format string, arg ...interface{}) int {
	str := fmt.Sprintf(format, arg...)
	//fmt.Printf("SLog:%s\n", str);
	cs := C.CString(str)
	defer C.free(unsafe.Pointer(cs))

	result := C.wrap_slog_log(C.int(sld), C.int(log_level), cs)
	return int(result)
}

/***
Wrapper Log of write bytes directly
@sld:opened slog descriptor
@log_level:refer to SL_XX.the level of this log.If log_level < filt_level(slog_open),it will not printed.
@RETVALUE:
-1:failed(err msg can be found in slog.log). 0:success
*/
func SLogBytes(sld int, log_level int, bs []byte) int {
	str := string(bs)
	//fmt.Printf("SLog:%s\n", str);
	cs := C.CString(str)
	defer C.free(unsafe.Pointer(cs))

	result := C.wrap_slog_log(C.int(sld), C.int(log_level), cs)
	return int(result)
}

/***
Wrapper Log of write string directly
@sld:opened slog descriptor
@log_level:refer to SL_XX.the level of this log.If log_level < filt_level(slog_open),it will not printed.
@RETVALUE:
-1:failed(err msg can be found in slog.log). 0:success
*/
func SLogString(sld int, log_level int, str string) int {
	cs := C.CString(str)
	defer C.free(unsafe.Pointer(cs))

	result := C.wrap_slog_log(C.int(sld), C.int(log_level), cs)
	return int(result)
}

/***
Change Attr
@sld:opened slog descriptor
@filt_level:refer to SLOG_LEVEL. If No Change Sets to -1.
@degree:refer to SLOG_DEGREE. If No Change sets to -1.
@size:Change log size. If No Change sets to -1.
@rotate:Change Max Rotate Number. If No Change sets to -1.
@format:Change format of single item. If No change sets to -1.
*/
func SLogChgAttr(sld int, filt_level int, degree int, size int, rotate int, format int) int {
	result := C.slog_chg_attr(C.int(sld), C.int(filt_level), C.int(degree), C.int(size), C.int(rotate), C.int(format))
	return int(result)
}
