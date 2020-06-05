package comm

import (
    "time"
    "syscall"
)

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
			    	    pconfig.ChInfo <- INFO_EXIT;
			    	case syscall.SIGUSR1:    
			    	    log.Info("recv signal usr1!");
			    	    pconfig.ChInfo <- INFO_USR1;
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