package main

import (
	// "flag"
	"os"
	"os/signal"
	"runtime"
	// "runtime/pprof"
	"syscall"
)

// var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func sigHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGHUP)

	for sig := range sigChan {
		info.Printf("%v caught, exit\n", sig)
		storeSiteStat()
		break
	}
	/*
		if *cpuprofile != "" {
			pprof.StopCPUProfile()
		}
	*/
	os.Exit(0)
}

var hasParentProxy = false

func main() {
	// Parse flags after load config to allow override options in config
	cmdLineConfig := parseCmdLineConfig()
	if cmdLineConfig.PrintVer {
		printVersion()
		os.Exit(0)
	}

	parseConfig(cmdLineConfig.RcFile)
	updateConfig(cmdLineConfig)

	initLog()
	initAuth()
	initSocksServer()
	initShadowSocks()
	initSiteStat()
	initPAC()

	if len(parentProxyCreator) == 0 {
		info.Println("no parent proxy server, can't handle blocked sites")
	} else {
		hasParentProxy = true
	}

	/*
		if *cpuprofile != "" {
			f, err := os.Create(*cpuprofile)
			if err != nil {
				info.Println(err)
				os.Exit(1)
			}
			pprof.StartCPUProfile(f)
		}
	*/

	if config.Core > 0 {
		runtime.GOMAXPROCS(config.Core)
	}

	go sigHandler()
	go runSSH()
	go runEstimateTimeout()

	done := make(chan byte, 1)
	// save 1 goroutine (a few KB) for the common case with only 1 listen address
	if len(config.ListenAddr) > 1 {
		for _, addr := range config.ListenAddr[1:] {
			go NewProxy(addr).Serve(done)
		}
	}
	NewProxy(config.ListenAddr[0]).Serve(done)
	for i := 0; i < len(config.ListenAddr); i++ {
		<-done
	}
}
