package comm

import (
	"os"
	"runtime"
	"runtime/pprof"
	"time"
    "syscall"
)

const (
	CPU_PROFILE string = "cpu.profile"
	MEM_PROFILE string = "mem.profile"
)

type ProfileConfig struct{
	Stat bool //false:closed true:open
	file_cpu * os.File
	file_mem * os.File
}



func HandleSignal(pconfig *CommConfig) {
	log := pconfig.Log;
	for {
			for v := range pconfig.ChSig {
				
			    switch v {
			    	case syscall.SIGINT:    
			    	    log.Info("recv signal int!");
			    	    pconfig.ChInfo <- INFO_EXIT;
			    	case syscall.SIGTERM:    
			    	    log.Info("recv signal term!");
			    	    pconfig.ChInfo <- INFO_PPROF;
			    	case syscall.SIGUSR1:    
			    	    log.Info("recv signal usr1!");
			    	    pconfig.ChInfo <- INFO_RELOAD_CFG;
			    	case syscall.SIGUSR2:    
			    	    log.Info("recv signal usr2!");
			    	    pconfig.ChInfo <- INFO_USR2;
			    	default:    
			    	    log.Info("unknown signal %v" , v);                
			    }

			}
		time.Sleep(time.Second);
	}
}


func DefaultHandleProfile(pconfig *CommConfig) bool {

    //start profile
    if !pconfig.PProf.Stat {
    	return StartPProf(pconfig);
	}

	//end profile
	return EndPProf(pconfig);
}

func CloseProFile(pconfig *CommConfig) {
	if pconfig.PProf.file_cpu != nil {
		pconfig.PProf.file_cpu.Close();
		pconfig.PProf.file_cpu = nil;
	}

	if pconfig.PProf.file_mem != nil {
		pconfig.PProf.file_mem.Close();
		pconfig.PProf.file_mem = nil;
	}

}


func StartPProf(pconfig *CommConfig) bool {
	var _func_ = "<StartPProf>";
	var err error
	log := pconfig.Log;
	profcfg := &pconfig.PProf;

	//check stat
	if profcfg.Stat {
		log.Err("%s failed! prof is opended!" , _func_);
		return false;
	}

    //open profile
    profcfg.file_cpu  , err = os.OpenFile(CPU_PROFILE , os.O_CREATE|os.O_RDWR|os.O_TRUNC , 0766);
    if err != nil {
        log.Err("%s open %s failed! err:%v" , _func_ , CPU_PROFILE , err);
        CloseProFile(pconfig);
        return false;
	}

	profcfg.file_mem  , err = os.OpenFile(MEM_PROFILE , os.O_CREATE|os.O_RDWR|os.O_TRUNC , 0766);
	if err != nil {
		log.Err("%s open %s failed! err:%v" , _func_ , MEM_PROFILE , err);
        CloseProFile(pconfig);
		return false;
	}


    //go>>>>>
    err = pprof.StartCPUProfile(profcfg.file_cpu);
    if err != nil {
        log.Err("%s start cpu profile failed! err:%v" , _func_ , err);
        CloseProFile(pconfig);
		return false;
	}

	profcfg.Stat = true;
    log.Info("%s starts success!" , _func_);
    return true;
}

func EndPProf(pconfig *CommConfig) bool {
	var _func_ = "<EndPProf>";
	var err error
	log := pconfig.Log;
	profcfg := &pconfig.PProf;

	if !profcfg.Stat {
		log.Err("%s not open!" , _func_);
		return false;
	}

    //stop cpu profile
    if profcfg.file_cpu != nil {
    	pprof.StopCPUProfile();
	}

    //stop mem profile
    if profcfg.file_mem != nil {
    	runtime.GC();
    	err = pprof.WriteHeapProfile(profcfg.file_mem);
    	if err != nil {
    		log.Err("%s write mem-file:%s failed! err:%v" , _func_ , MEM_PROFILE , err);
		}
	}

    //close
    profcfg.Stat = false;
    CloseProFile(pconfig);
	log.Info("%s finish!" , _func_);
	return true;
}