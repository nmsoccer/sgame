package lib

import (
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"sgame/servers/comm"
	"strconv"
	"time"
)

const (
	TIME_FORMAT_SEC = "2006-01-02 15:04:05"
	INDEX_TMPL      = "./html_tmpl/index.html"
	DETAIL_TMPL     = "./html_tmpl/detail.html"
)

type PanelServ struct {
	config *Config
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
func (pserv *PanelServ) index_handle(w http.ResponseWriter, r *http.Request) {
	var _func_ = "panel.detail_get"
	log := pserv.config.Comm.Log
	pconfig := pserv.config

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
	var _func_ = "panel.detail_get"
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

	//output
	tmpl.Execute(w, pconfig.WatchMap[proc_id])

}

func (pserv *PanelServ) detail_handle(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {
		pserv.detail_get(w, r)
	}

}

func (pserv *PanelServ) cmd_handle(w http.ResponseWriter , r *http.Request) {
	var _func_ = "<cmd_handle>"
	pconfig := pserv.config;
	log := pconfig.Comm.Log;

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
	if operation != "reload_cfg" && operation != "reload_table" {
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
