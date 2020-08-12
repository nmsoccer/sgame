package main

/*
*SPush is Created by nmsoccer
* more instruction could be found @https://github.com/nmsoccer/spush
 */
import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var DefaultPushTimeout int = 30 //default 30s timeout
var TransProto = "udp"
var ListenPort int = 32808                      // dispatcher listen port
var ListenAddr = ":" + strconv.Itoa(ListenPort) //dispatcher listen addr

var ConfFile string = "./conf.json" //default conf file
var CfgDir string = "./cfg/"
var tmpl_map map[string]string = make(map[string]string)

//options
var PushAll = flag.Bool("P", false, "push all procs")
var PushSome = flag.String("p", "", "push some procs")
var CreateCfg = flag.Bool("C", false, "just create cfg")
var Verbose = flag.Bool("v", false, "verbose")
var SConfFile = flag.String("f", "", "spec conf file,default using ./conf.json")
var RemainFootPrint = flag.Bool("r", false, "remain footprint at every deployed dir")
var OnlyCfg = flag.Bool("O", false, "only push cfg")
var OnlyBin = flag.Bool("o", false, "only push bin")

type Proc struct {
	Name    string
	Bin     []string
	Host    string
	HostDir string `json:"host_dir"`
	//CopyCfg int `json:"copy_cfg"`
	Cmd string `json:"cmd"`
}

type ProcCfg struct {
	Name      string
	CfgName   string `json:"cfg_name"`
	CfgTmpl   string `json:"cfg_tmpl"`
	TmplParam string `json:"tmpl_param"`
}

type Conf struct {
	TaskName      string `json:"task"`
	DeployHost    string `json:"deploy_host"`
	DeployTimeOut int    `json:"deploy_timeout"`
	DeployUser    string `json:"remote_user"`
	DeployPass    string `json:"remote_pass"`
	Procs         []*Proc
	ProcCfgs      []*ProcCfg `json:"proc_cfgs"`
}

type MProc struct {
	proc     *Proc
	cfg_file string
}

const (
	PUSH_ING int = iota
	PUSH_SUCCESS
	PUSH_ERR
)

type TransMsg struct {
	Mtype   int    `json:"msg_type"` //1:report 2:response
	Mproc   string `json:"msg_proc"`
	Mresult int    `json:"msg_result"` //refer const PUSH_XX
	Minfo   string `json:"msg_info"`
}

type PushResult struct {
	proc   *Proc
	status int //refer const PUSH_XX
	info   string
}

type PushTotal struct {
	complete int
	push_map map[string]*PushResult
}

var conf Conf
var proc_map = make(map[string]*MProc)
var push_total = PushTotal{complete: 0}
var push_lock sync.Mutex

func main() {
	var curr_time time.Time
	curr_time = time.Now()
	fmt.Printf("\n++++++++++++++++++++spush (%04d-%02d-%02d %02d:%02d:%02d)++++++++++++++++++++\n", curr_time.Year(), int(curr_time.Month()),
		curr_time.Day(), curr_time.Hour(), curr_time.Minute(), curr_time.Second())

	flag.Parse()
	//check flag
	if flag.NFlag() <= 0 {
		flag.PrintDefaults()
		return
	}

	if *SConfFile != "" && len(*SConfFile) > 0 {
		ConfFile = *SConfFile
	}

	if *OnlyCfg && *OnlyBin {
		flag.PrintDefaults()
		fmt.Printf("-o and -O only one option allowed! if both false , default push both!\n")
		return
	}

	//open conf
	file, err := os.Open(ConfFile)
	if err != nil {
		fmt.Printf("open %s failed! err:%v", ConfFile, err)
		return
	}
	defer file.Close()

	//decode
	var decoder *json.Decoder
	decoder = json.NewDecoder(file)
	err = decoder.Decode(&conf)
	if err != nil {
		fmt.Printf("decode failed! err:%s", err)
		return
	}
	if len(conf.Procs) <= 0 {
		fmt.Printf("empty proc! nothing to do\n")
		return
	}

	//check arg
	if conf.TaskName == "" || len(conf.TaskName) <= 0 {
		fmt.Printf("conf.task not set ! please check\n")
		return
	}

	//set default
	if conf.DeployHost == "" {
		conf.DeployHost = "127.0.0.1"
	}

	if conf.DeployTimeOut == 0 {
		conf.DeployTimeOut = DefaultPushTimeout //default 60s
	}

	if conf.DeployUser == "" {
		conf.DeployUser = "#"
	}

	if conf.DeployPass == "" {
		conf.DeployPass = "#"
	}

	//pp
	if !*Verbose || *Verbose {
		go func() {
			for {
				fmt.Printf(".")
				time.Sleep(1e9) //1s
			}
		}()
	}

	//mproc
	var mproc *MProc
	for _, proc := range conf.Procs {
		//convert bin file path to abs
		if len(proc.Bin) > 0 {
			for i, file_path := range proc.Bin {
				abs_path, err := filepath.Abs(file_path)
				if err != nil {
					fmt.Printf("Convert %s to abs Path Failed! Please Check proc <%s>\n ", file_path, proc.Name)
					return
				}
				proc.Bin[i] = abs_path
			}
		}

		if proc.HostDir == "" || len(proc.HostDir) <= 0 {
			fmt.Printf("proc.host_dir Not Set! Please Check proc <%s>\n", proc.Name)
			return
		}
		//convert to abs path
		abs_host_dir, err := filepath.Abs(proc.HostDir)
		if err != nil {
			fmt.Printf("Convert Dir to abs Path Failed! Please Check proc <%s>\n ", proc.Name)
			return
		}
		proc.HostDir = abs_host_dir

		if proc.Host == "" || len(proc.Host) <= 0 { //default set local
			proc.Host = "127.0.0.1"
		}

		if proc.Cmd == "" || len(proc.Cmd) <= 0 { //default cmd nop
			proc.Cmd = ":"
		}

		mproc = new(MProc)
		mproc.proc = proc
		proc_map[proc.Name] = mproc
		v_print("proc:%s bin:%v host:%s host_dir:%s\n", proc.Name, mproc.proc.Bin, mproc.proc.Host, mproc.proc.HostDir)
	}
	v_print("proc_map:%v\n", proc_map)

	//handle option
	switch {
	case *CreateCfg:
		fmt.Println("create cfg...")
		create_all_cfg()
		//fallthrough;
	case *PushAll:
		fmt.Println("push all procs")
		push_procs(conf.Procs)
		break
	case *PushSome != "":
		fmt.Printf("push some procs:%s\n", *PushSome)
		//1. match procs
		procs := parse_some_procs_arg(*PushSome)
		fmt.Printf("matched procs num:%d\n", len(procs))
		if len(procs) <= 0 {
			fmt.Printf("no-proc found!\n")
			break
		}

		//2. push
		push_procs(procs)
		break
	default:
		fmt.Println("nothing to do")
		break
	}

	curr_time = time.Now()
	fmt.Printf("\n+++++++++++++++++++++end (%04d-%02d-%02d %02d:%02d:%02d)+++++++++++++++++++++\n", curr_time.Year(), int(curr_time.Month()),
		curr_time.Day(), curr_time.Hour(), curr_time.Minute(), curr_time.Second())
	return
}

//src like "key=value , key2=value2 , ..."
func parse_tmpl_param2(src string, result map[string]string) int {
	if src == "" {
		return -1
	}

	//split ","
	str_list := strings.Split(src, ",")
	fmt.Printf("str_list:%v\n", str_list)

	//split "="
	for _, item := range str_list {
		k_v := strings.Split(item, "=")
		v_print("key=%s value=%s\n", k_v[0], k_v[1])
		result[k_v[0]] = k_v[1]
	}
	fmt.Printf("result:%v\n", result)
	return 0
}

func parse_tmpl_param(src string, result map[string]string) int {
	if src == "" {
		return -1
	}

	//split ","
	org := []byte(src)
	bytes_list := bytes.Split(org, []byte(","))

	//split "=" (item like "key=value")
	for _, item := range bytes_list {
		k_v := bytes.Split(item, []byte("="))

		k_v[0] = bytes.Trim(k_v[0], " ")
		k_v[1] = bytes.Trim(k_v[1], " ")
		result[string(k_v[0])] = string(k_v[1])
	}
	//fmt.Printf("result:%v\n", result);
	return 0
}

//后缀的匹配
func parse_some_procs_arg(pattern string) []*Proc {
	var procs = make([]*Proc, 0)

	v_print("pattern:%s\n", pattern)
	//try-match all proc
	var pproc *Proc

	for _, pproc = range conf.Procs {
		ok, _ := regexp.MatchString(pattern, pproc.Name)
		if ok {
			v_print("%s matched!\n", pproc.Name)
			procs = append(procs, pproc)
		}
	}

	return procs
}

func push_procs(procs []*Proc) {
	//1. create cfg
	create_all_cfg()

	//2. routine
	ch := make(chan string)
	init_push_result(procs)
	go check_push_result(ch)
	timing_ch := time.Tick(time.Second * time.Duration(conf.DeployTimeOut)) //default 30s timeout

	//3. gen pkg
	var pproc *Proc
	for _, pproc = range procs {
		go gen_pkg(pproc)
	}

	//4. push result
	select {
	case <-timing_ch:
		fmt.Printf("\n----------Push <%s> Timeout----------\n", conf.TaskName)

	case push_result := <-ch:
		fmt.Printf("\n----------Push <%s> Result---------- \n%s\n", conf.TaskName, push_result)
	}
	time.Sleep(1e9)
	print_push_result()
}

func gen_pkg(pproc *Proc) int {
	var _func_ = "<gen_pkg>"

	//pkg-dir
	//curr_time := time.Now();
	var pkg_dir = ""
	pkg_dir = fmt.Sprintf("./pkg/%s/%s/", conf.TaskName, pproc.Name)
	//pkg_dir = fmt.Sprintf("./pkg/%s/%d-%02d-%02d/" , pproc.Name , curr_time.Year() , int(curr_time.Month()) , curr_time.Day());
	//fmt.Println(pkg_dir);

	//rm exist dir
	err := os.RemoveAll(pkg_dir)
	if err != nil {
		fmt.Printf("%s remove old dir:%s failed! err:%s\n", _func_, pkg_dir, err)
		return -1
	}

	//create dir
	err = os.MkdirAll(pkg_dir, 0766)
	if err != nil {
		fmt.Printf("%s create dir %s failed! err:%s\n", _func_, pkg_dir, err)
		return -1
	}

	//copy files
	cp_arg := []string{"-Rf"}

	if *OnlyBin { //only bin files
		cp_arg = append(cp_arg, pproc.Bin...)
	} else if *OnlyCfg { //only cfg
		cp_arg = append(cp_arg, proc_map[pproc.Name].cfg_file)
	} else { //both
		cp_arg = append(cp_arg, pproc.Bin...)
		if proc_map[pproc.Name].cfg_file != "" { //copy cfg
			cp_arg = append(cp_arg, proc_map[pproc.Name].cfg_file)
		}
	}

	cp_arg = append(cp_arg, pkg_dir)
	v_print("exe cp %v\n", cp_arg)

	cp_cmd := exec.Command("cp", cp_arg...)
	output_info := bytes.Buffer{}
	cp_cmd.Stdout = &output_info
	err = cp_cmd.Run()
	if err != nil {
		fmt.Printf("cp failed! err:%s cmd:%v\n", err, cp_cmd.Args)
		push_lock.Lock()
		push_total.complete += 1
		push_total.push_map[pproc.Name].status = PUSH_ERR
		push_total.push_map[pproc.Name].info = "copy failed:" + err.Error()
		push_lock.Unlock()

		return -1
	}

	//exe tool
	var keep_footprint = "n"
	if *RemainFootPrint {
		keep_footprint = "y"
	}

	push_cmd := exec.Command("./tools/push.sh", conf.TaskName, pproc.Name, pproc.Host, pproc.HostDir, conf.DeployHost, strconv.Itoa(ListenPort),
		conf.DeployUser, conf.DeployPass, keep_footprint, pproc.Cmd, "&")
	cmd_result := bytes.Buffer{}
	push_cmd.Stdout = &cmd_result
	v_print("exe push %v\n", push_cmd.Args)
	err = push_cmd.Run()
	if err != nil {
		fmt.Printf("exe push failed! err:%s cmd:%v\n", err, push_cmd.Args)
		push_lock.Lock()
		push_total.complete += 1
		push_total.push_map[pproc.Name].status = PUSH_ERR
		push_total.push_map[pproc.Name].info = "dispatch failed:" + err.Error()
		push_lock.Unlock()
		v_print("complete:%d <%s>\n", push_total.complete, pproc.Name)
		return -1
	}
	if *Verbose {
		fmt.Printf("%s\n", cmd_result.String())
	}
	return 0
}

func create_all_cfg() {
	var success = 0
	var pcfg *ProcCfg
	//del old dir
	old_dir := CfgDir + conf.TaskName + "/"
	err := os.RemoveAll(old_dir)
	if err != nil {
		fmt.Printf("remove %s failed! err:%s\n", old_dir, err)
	}

	for _, pcfg = range conf.ProcCfgs {
		ret := create_cfg(pcfg)
		if ret == 0 {
			success += 1
		}
	}
	fmt.Printf("create cfg:%d/%d\n", success, len(conf.ProcCfgs))
}

func create_cfg(pcfg *ProcCfg) int {
	cfg_base := CfgDir + conf.TaskName + "/" + pcfg.Name //cfg/$task/$proc_name/
	cfg_path := cfg_base
	//parse CfgName
	good_path := strings.Trim(pcfg.CfgName, "/")
	sub_dir, cfg_file := filepath.Split(good_path)

	//create dir
	if len(sub_dir) > 0 {
		cfg_path = cfg_path + "/" + sub_dir
	} else {
		cfg_path += "/"
	}
	err := os.MkdirAll(cfg_path, 0766)
	if err != nil {
		fmt.Printf("create dir %s failed! err:%s", cfg_path, err)
		return -1
	}

	//create file
	//cfg_real := cfg_path + "/" + pcfg.CfgName;
	cfg_real := cfg_path + cfg_file
	fp, err := os.OpenFile(cfg_real, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("open %s failed! err:%s", cfg_real, err)
		return -1
	}
	defer fp.Close()

	//read tmpl content
	if pcfg.CfgTmpl != "" && tmpl_map[pcfg.CfgTmpl] == "" {
		tmp_fp, err := os.Open(pcfg.CfgTmpl)
		if err != nil {
			fmt.Printf("open %s failed! err:%s", pcfg.CfgTmpl, err)
			return -1
		}
		defer tmp_fp.Close()

		//read
		var content []byte = make([]byte, 2048)
		n, err := tmp_fp.Read(content)
		if err != nil {
			fmt.Printf("read %s failed! err:%s", pcfg.CfgTmpl, err)
			return -1
		}
		tmpl_map[pcfg.CfgTmpl] = string(content[:n])
	}

	//parse tmpl param
	cfg_content := []byte(tmpl_map[pcfg.CfgTmpl])
	if pcfg.TmplParam != "" {
		tmpl_param_map := make(map[string]string)
		res := parse_tmpl_param(pcfg.TmplParam, tmpl_param_map)
		if res != 0 {
			fmt.Printf("parse tmpl param failed! cfg:%s tmpl:%s param:%s", cfg_real, pcfg.CfgTmpl, pcfg.TmplParam)
			return -1
		}

		//replace param
		for k, v := range tmpl_param_map {
			//cfg_content = bytes.ReplaceAll(cfg_content, []byte("$"+k), []byte(v)); not avai for go < 1.12
			cfg_content = bytes.Replace(cfg_content, []byte("$"+k), []byte(v), -1)
		}

	}
	//fmt.Printf("after parsing:%s\n len:%d\n", string(cfg_content) , len(cfg_content));

	//write to cfg
	_, err = fp.Write(cfg_content[:len(cfg_content)])
	if err != nil {
		fmt.Printf("write to %s failed! err:%s", cfg_real, err)
		return -1
	}
	if len(sub_dir) > 0 {
		first_dir := bytes.Split([]byte(sub_dir), []byte("/"))[0]
		proc_map[pcfg.Name].cfg_file = cfg_base + "/" + string(first_dir) + "/"
	} else {
		proc_map[pcfg.Name].cfg_file = cfg_real
	}
	v_print("create %s success!\n", cfg_real)
	return 0
}

func init_push_result(procs []*Proc) {
	//push_map := make(map[string] *PushResult);
	push_total.push_map = make(map[string]*PushResult)
	push_total.complete = 0

	for _, pproc := range procs {
		push_total.push_map[pproc.Name] = &PushResult{status: PUSH_ING, info: "dipatching", proc: pproc}
	}
}

func print_push_result() {
	check_map := push_total.push_map
	code_converse := map[int]string{PUSH_ING: "timeout", PUSH_SUCCESS: "success", PUSH_ERR: "err"}
	results := make([]string, len(check_map))
	i := 0
	success := 0
	push_lock.Lock()
	for proc_name, presult := range check_map {
		str := fmt.Sprintf("[%s]::%s %s", proc_name, code_converse[presult.status], presult.info)
		results[i] = str
		i += 1
		if presult.status == PUSH_SUCCESS {
			success += 1
		}
	}
	push_lock.Unlock()
	fmt.Printf("\n[%d/%d]\n", success, len(check_map))
	for _, result := range results {
		fmt.Printf("%s\n", result)
	}
	/*
		for proc_name , presult := range check_map {
			fmt.Printf("[%s]::%s %s\n", proc_name , code_converse[presult.status] , presult.info);
		}
	*/
}

func check_push_result(c chan string) {
	check_map := push_total.push_map
	result := "ok"
	var conn *net.UDPConn

	//construct check map
	v_print("map len:%d and map:%v\n", len(check_map), check_map)

	//check complete
	go func() { //这里启动一个routine 来检查complete的情况
		for {
			if push_total.complete >= len(push_total.push_map) {
				v_print("push from check-routine complete!\n")
				c <- ""
				break
			}
			time.Sleep(1e9) //1s
		}
	}()

	//resolve addr
	my_addr, err := net.ResolveUDPAddr(TransProto, ListenAddr)
	if err != nil {
		fmt.Printf("resolve  %s failed! we may not recv push results! err:%s", ListenAddr, err)
		result = "create listener failed by addr fault"
		goto _end
	}

	//listen and response
	conn, err = net.ListenUDP(TransProto, my_addr)
	if err != nil {
		fmt.Printf("listen  %s failed! we may not recv push results! err:%s", ListenAddr, err)
		result = "create listener failed by listen fault "
		goto _end
	}

	//handle
	for {
		//check complete
		if push_total.complete >= len(check_map) {
			result = "ok"
			break
		}

		//read pkg
		recv_buff := make([]byte, 256)
		n, peer_addr, err := conn.ReadFromUDP(recv_buff)
		if err != nil {
			fmt.Printf("recv from udp failed! err:%s\n", err)
			time.Sleep(1e9) //1s
			continue
		}

		//print pkg
		recv_buff = recv_buff[:n]
		v_print("recv from %s msg:%s\n", peer_addr.String(), string(recv_buff))

		//decode
		var msg TransMsg
		err = json.Unmarshal([]byte(recv_buff), &msg)
		if err != nil {
			fmt.Printf("json decode failed! err:%s\n", err)
			continue
		}
		v_print("msg:%v\n", msg)

		//check
		if msg.Mtype != 1 {
			fmt.Printf("mst type illegal! type:%d\n", msg.Mtype)
			continue
		}
		if check_map[msg.Mproc] == nil {
			fmt.Printf("msg proc illegal! proc:%s\n", msg.Mproc)
			continue
		}

		//response
		conn.WriteTo([]byte("good night"), peer_addr)

		//set status
		if check_map[msg.Mproc].status == PUSH_ING {
			push_lock.Lock()
			check_map[msg.Mproc].status = msg.Mresult
			check_map[msg.Mproc].info = msg.Minfo
			push_total.complete += 1
			push_lock.Unlock()
		}

	}

_end:
	c <- result
}

func v_print(format string, ext_arg ...interface{}) {
	if *Verbose {
		fmt.Printf(format, ext_arg...)
	}
}
