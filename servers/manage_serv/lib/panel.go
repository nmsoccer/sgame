package lib

import (
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"sgame/servers/comm"
	"strconv"
	"strings"
	"time"
)

const (
	TIME_FORMAT_SEC = "2006-01-02 15:04:05"
	INDEX_TMPL      = "./html_tmpl/index.html"
	DETAIL_TMPL     = "./html_tmpl/detail.html"
	LOGIN_TMPL      = "./html_tmpl/login.html"

	COOKIE_NAME     = "manage_token"
)

type AuthInfo struct{
	name string
	pass string
	login int64
	token string
}


type PanelServ struct {
	config *Config
	token_seq uint16
}

func ParseAuth(pconfig *Config) bool {
	var _func_ = "<ParseAuth>";
	log := pconfig.Comm.Log;

	//reclear
	pconfig.AuthMap = make(map[string]*AuthInfo);
    pconfig.TokenMap = make(map[string]string);

    //parse
    for _ , info := range pconfig.FileConfig.Auth {
    	basic := strings.Split(info , ":");
    	if len(basic) != 2 {
    		log.Err("%s parse %s failed! please check!" , _func_ , info);
    		return false;
		}

    	name := basic[0];
    	pass := basic[1];
    	pauth := new(AuthInfo);
    	pconfig.AuthMap[name] = pauth;
    	pauth.name = name;
    	pauth.pass = pass;
	}

    log.Info("%s done! auth:%v" , _func_ , pconfig.AuthMap);
    return true;
}




func StartPanel(pconfig *Config) *PanelServ {
	var _func_ = "<StartPanel>"
	log := pconfig.Comm.Log

	//new
	pserv := new(PanelServ)
	if pserv == nil {
		log.Err("%s fail! new serv failed!", _func_)
		return nil
	}

	//set
	pserv.config = pconfig
	pserv.serve()
	return pserv
}

/*----------------------STATIC FUNC--------------------*/
func (pserv *PanelServ) check_auth(w http.ResponseWriter , r *http.Request) bool {
	var _func_ = "<panel.check_auth>";
	pconfig := pserv.config;
	log := pconfig.Comm.Log;

	//GET COOKIE
	cookie , err := r.Cookie(COOKIE_NAME);
    if err != nil {
    	log.Err("%s get cookie failed! err:%v" , _func_ , err);
    	http.Redirect(w , r , "/login" , http.StatusUnauthorized);
    	return false;
	}

	//check token
	token := cookie.Value;
	name , ok := pconfig.TokenMap[token];
	if !ok {
		log.Err("%s token not found! token:%v" , _func_ , token);
		http.Redirect(w , r , "/login" , http.StatusUnauthorized);
		return false;
	}

	//check name
    pauth , ok := pconfig.AuthMap[name];
    if !ok {
		log.Err("%s user illegal! name:%s" , _func_ , name);
		http.Redirect(w , r , "/login" , http.StatusUnauthorized);
		return false;
	}


    //check auth info
    curr_ts := time.Now().Unix();
    if pauth.login + int64(pconfig.FileConfig.AuthExpire) < curr_ts {
		log.Err("%s user expired! name:%s" , _func_ , name);
		http.Redirect(w , r , "/login" , http.StatusUnauthorized);
		return false;
	}


    log.Info("%s passed %s" , _func_ , name);
    return true;
}



func (pserv *PanelServ) check_login(w http.ResponseWriter , r *http.Request) bool {
	var _func_ = "<panel.check_login>";
	pconfig := pserv.config;
	log := pconfig.Comm.Log;

	//check name
	name := r.FormValue("name");
    if name == "" {
    	log.Err("%s fail! name empty!" , _func_);
    	fmt.Fprintf(w , "name emtpy!");
    	return false;
	}

	//name legal
	auth_info , ok := pconfig.AuthMap[name];
	if !ok {
		fmt.Fprintf(w , "name invalid!");
		return false;
	}

	//check pass
	pass := r.FormValue("pass");
	if name == "" {
		log.Err("%s fail! pass empty!" , _func_);
		fmt.Fprintf(w , "pass error!");
		return false;
	}

	//pass match
	if auth_info.pass != pass {
		log.Err("%s fail! pass not match!" , _func_);
		fmt.Fprintf(w , "pass error!");
		return false;
	}

    return true;
}

func (pserv *PanelServ) login_get(w http.ResponseWriter , r *http.Request) {
	var _func_ = "<panel.login_get>"
	log := pserv.config.Comm.Log


	//template
	tmpl, err := template.ParseFiles(LOGIN_TMPL)
	if err != nil {
		log.Err("%s parse template %s failed! err:%v", _func_, LOGIN_TMPL, err)
		fmt.Fprintf(w, "parse error!")
		return
	}

	//output
	tmpl.Execute(w, nil);
}

func (pserv *PanelServ) login_post(w http.ResponseWriter , r *http.Request) {
	var _func_ = "<panel.login_post>"
	pconfig := pserv.config;
	log := pconfig.Comm.Log;

    if !pserv.check_login(w , r) {
    	return;
	}
	user_name := r.FormValue("name");
	log.Info("%s login success! name:%s" , _func_ , user_name);


	//token
    token := fmt.Sprintf("%d" , comm.GenerateLocalId(int16(pconfig.ProcId) , &pserv.token_seq));
    if pconfig.TokenMap == nil {
    	pconfig.TokenMap = make(map[string]string);
	}
    pconfig.TokenMap[token] = user_name;

    //update auth
    pconfig.AuthMap[user_name].token = token;
    pconfig.AuthMap[user_name].login = time.Now().Unix();


	//set cookie
	var cookie http.Cookie;
	cookie.Name = COOKIE_NAME;
	cookie.Value = token;
	cookie.Expires = time.Now().Add(time.Second * time.Duration(pconfig.FileConfig.AuthExpire));
    http.SetCookie(w , &cookie);


	http.Redirect(w , r , "/index" , http.StatusFound);

}

func (pserv *PanelServ) login_handle(w http.ResponseWriter , r *http.Request) {
    if r.Method == "GET" {
    	pserv.login_get(w , r);
	} else if r.Method == "POST" {
		pserv.login_post(w , r);
	} else {
		fmt.Fprintf(w , "invalid method!");
	}
}


func (pserv *PanelServ) index_handle(w http.ResponseWriter, r *http.Request) {
	var _func_ = "<panel.index_handle>"
	log := pserv.config.Comm.Log
	pconfig := pserv.config

	//check auth
    if !pserv.check_auth(w , r) {
    	return;
	}

	//template
	tmpl, err := template.ParseFiles(INDEX_TMPL)
	if err != nil {
		log.Err("%s parse template %s failed! err:%v", _func_, INDEX_TMPL, err)
		fmt.Fprintf(w, "parse error!")
		return
	}

	//output
	tmpl.Execute(w, pconfig.WatchMap)

}

func (pserv *PanelServ) detail_get(w http.ResponseWriter, r *http.Request) {
	var _func_ = "<panel.detail_get>"
	log := pserv.config.Comm.Log
	pconfig := pserv.config

	//get proc id
	proc_id, err := strconv.Atoi(r.FormValue("proc"))
	if err != nil {
		log.Err("%s no proc_id found!", _func_)
		fmt.Fprintf(w, "not found!")
		return
	}

	//template
	tmpl, err := template.ParseFiles(DETAIL_TMPL)
	if err != nil {
		log.Err("%s parse template %s failed! err:%v", _func_, DETAIL_TMPL, err)
		fmt.Fprintf(w, "parse error!")
		return
	}

	//modify some info
	if len(pconfig.WatchMap[proc_id].Stat.MonitorInfo) > 0 {
		monitor_info := string(pconfig.WatchMap[proc_id].Stat.MonitorInfo);
		monitor_info = strings.Replace(monitor_info, "\n", "<br/>", -1);
		monitor_info = strings.Replace(monitor_info, " ", "&nbsp", -1);
		pconfig.WatchMap[proc_id].Stat.MonitorInfo = template.HTML(monitor_info);
	}

	//output
	tmpl.Execute(w, pconfig.WatchMap[proc_id])

}

func (pserv *PanelServ) detail_handle(w http.ResponseWriter, r *http.Request) {
    if !pserv.check_auth(w , r) {
    	return;
	}

	if r.Method == "GET" {
		pserv.detail_get(w, r)
	}

}

func (pserv *PanelServ) cmd_handle(w http.ResponseWriter , r *http.Request) {
	var _func_ = "<panel.cmd_handle>"
	pconfig := pserv.config;
	log := pconfig.Comm.Log;

	//check auth
	if !pserv.check_auth(w , r) {
		return;
	}

	//get proc name
	proc_name := r.FormValue("proc_name");
	operation := r.FormValue("operation");

    //check
    if proc_name=="" || operation=="" {
    	log.Err("%s arg not full!" , _func_);
    	fmt.Fprintf(w , "ProcName and Operation must fill!");
    	return;
	}

	//operation
	if operation != comm.RELOAD_CMD_CFG && operation != comm.RELOAD_CMD_TAB {
		log.Err("%s illegal operation:%s" , _func_ , operation);
		fmt.Fprintf(w , "Operation illegal!");
		return;
	}


	//regular
	proc_exp , err := regexp.Compile(proc_name);
	if err != nil {
		log.Err("%s exp compile failed! proc_name:%s err:%v" , _func_ , proc_name , err);
		fmt.Fprintf(w  , "proc_name regexp illegal!");
		return;
	}

	fmt.Fprintf(w , "proc_name:%s operation:%s Handling...\n" , proc_name , operation);
	//match proc
	for name , proc_id := range pconfig.Name2Id {
		if ! proc_exp.MatchString(name) {
			continue;
		}

		//matched
		fmt.Fprintf(w , "%s of %d in progressing...\n" , name , proc_id);

		//send msg
		curr_ts := time.Now().Unix();
		pconfig.WatchMap[proc_id].Stat.ReloadTime = time.Now();
		pconfig.WatchMap[proc_id].Stat.ReloadStat = comm.RELOAD_STAT_ING;
		pconfig.WatchMap[proc_id].Stat.ReloadCmd = operation;
        pserv.report_cmd(proc_id , comm.REPORT_PROTO_CMD_RELOAD , curr_ts , operation , nil);
	}



    log.Info("%s proc_name:%s operation:%s" , _func_ , proc_name , operation);
}



func (pserv *PanelServ) serve() {
	var _func_ = "panelserv.serve"
	log := pserv.config.Comm.Log

	go func() {
		http.Handle("/", http.HandlerFunc(pserv.index_handle))
		http.Handle("/detail", http.HandlerFunc(pserv.detail_handle))
        http.Handle("/cmd" , http.HandlerFunc(pserv.cmd_handle));
		http.Handle("/login" , http.HandlerFunc(pserv.login_handle));
		err := http.ListenAndServe(pserv.config.FileConfig.HttpAddr, nil)
		if err != nil {
			log.Err("%s failed! err:%v", _func_, err)
			return
		}

	}()

}

/*
* Report Cmd To Normal Server
* @proto:Refer REPORT_PROTO_XX
* @v_msg: used for complex information. should be defined in report_proto.go
 */
func (pserv *PanelServ) report_cmd(proc_id int, proto int, v_int int64, v_str string, v_msg interface{}) bool {
	var _func_ = "<report_cmd>"
	pconfig := pserv.config
	log := pconfig.Comm.Log

	//get target
	ptarget, ok := pconfig.WatchMap[proc_id]
	if !ok {
		log.Err("%s target not found! proc_id:%d proto:%d", _func_, proc_id, proto)
		return false
	}

	//get addr
	if ptarget.Stat.addr == nil {
		log.Err("%s addr not set! target:%s proto:%d", _func_, ptarget.ProcName, proto)
		return false
	}

	//report msg
	var pmsg = new(comm.ReportMsg)
	pmsg.ProtoId = proto
	pmsg.ProcId = proc_id;
	pmsg.IntValue = v_int
	pmsg.StrValue = v_str
	pmsg.Sub = v_msg

	//send to ch
	pconfig.Recver.cmd_ch <- pmsg;
	return true
}
