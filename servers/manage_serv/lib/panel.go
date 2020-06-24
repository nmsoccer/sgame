package lib

import (
	"net/http"
	"fmt"
	"strconv"
	"html/template"
)

const (
	TIME_FORMAT_SEC="2006-01-02 15:04:05"
	INDEX_TMPL="./html_tmpl/index.html"
	DETAIL_TMPL="./html_tmpl/detail.html"
)


type PanelServ struct {
	config *Config
}


func StartPanel(pconfig *Config) *PanelServ {
	var _func_ = "<StartPanel>";
	log := pconfig.Comm.Log;
	
	//new
	pserv := new(PanelServ);
	if pserv == nil {
		log.Err("%s fail! new serv failed!" , _func_);
		return nil;
	}
	
	//set
	pserv.config = pconfig;
	pserv.serve();
	return pserv;
}

/*----------------------STATIC FUNC--------------------*/
func (pserv *PanelServ) index_handle(w http.ResponseWriter , r *http.Request) { 
    var _func_ = "panel.detail_get"
	log := pserv.config.Comm.Log;
	pconfig := pserv.config;
	
	//template
	tmpl , err := template.ParseFiles(INDEX_TMPL);
	if err != nil {
		log.Err("%s parse template %s failed! err:%v" , _func_ , INDEX_TMPL , err);
		fmt.Fprintf(w, "parse error!");
		return;
	}
	
	//output
	tmpl.Execute(w, pconfig.WatchMap);
		
}
/*
func (pserv *PanelServ) index_handle(w http.ResponseWriter , r *http.Request) {
	fmt.Fprintf(w, "<head><title>index </title></head><body bgcolor='white'>Servers<br/>");
	curr_ts := time.Now().Unix();
	for _ , pwatch := range pserv.config.WatchMap {
		fmt.Fprintf(w, "<a href='./detail?proc=%d'>[%s]</a> <br/>",  pwatch.proc_id , pwatch.ProcName);
		//start
		if pwatch.Stat.StartTime.Before(time.Unix(0, 0))  {
			fmt.Fprintf(w, "%s<font color='red'>", START_LABEL)
		} else {
			fmt.Fprintf(w, "%s<font color='green'>", START_LABEL)
		}
		fmt.Fprintf(w, "%v</font> ", pwatch.Stat.StartTime.Format(TIME_FORMAT_SEC));
		
		//hearbeat
		if pwatch.Stat.HeartBeat.Before(time.Unix(0, 0)) || pwatch.Stat.HeartBeat.Before(time.Unix(curr_ts-20 , 0)) {
			fmt.Fprintf(w, "%s<font color='red'>", HEART_LABEL)
		} else {
			fmt.Fprintf(w, "%s<font color='green'>", HEART_LABEL)
		}
		fmt.Fprintf(w, "%v</font> ", pwatch.Stat.HeartBeat.Format(TIME_FORMAT_SEC));
			
		fmt.Fprintf(w, "<br/>");
		
    }
	fmt.Fprintf(w, "</body>");
}*/


func (pserv *PanelServ) detail_get(w http.ResponseWriter , r *http.Request) {
	var _func_ = "panel.detail_get"
	log := pserv.config.Comm.Log;
	pconfig := pserv.config;
	
	//get proc id
	proc_id , err := strconv.Atoi(r.FormValue("proc"));
	if err != nil {
	    log.Err("%s no proc_id found!" , _func_);
	    fmt.Fprintf(w, "not found!");
	    return;
	}
	
	//template
	tmpl , err := template.ParseFiles(DETAIL_TMPL);
	if err != nil {
		log.Err("%s parse template %s failed! err:%v" , _func_ , DETAIL_TMPL , err);
		fmt.Fprintf(w, "parse error!");
		return;
	}
	
	//output
	tmpl.Execute(w, pconfig.WatchMap[proc_id]);
			
}


func (pserv *PanelServ) detail_handle(w http.ResponseWriter , r *http.Request) {
		
	if r.Method == "GET" {
	    pserv.detail_get(w, r);
	}
        
}


func (pserv *PanelServ) serve() {
	var _func_ = "panelserv.serve";
	log := pserv.config.Comm.Log;
	
    go func() {
    	http.Handle("/", http.HandlerFunc(pserv.index_handle));
    	http.Handle("/detail", http.HandlerFunc(pserv.detail_handle));
    	
    	err := http.ListenAndServe(pserv.config.FileConfig.HttpAddr, nil);
    	if err != nil {
    		log.Err("%s failed! err:%v" , _func_ , err);
    		return;
    	}
    	
    	
    }();	
	
}