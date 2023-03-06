package main

import (
	"log"
	"os"
	"runtime/pprof"

	"github.com/andreistan26/sync/src/cmd"
)

func main() {
	// enable cpu progiling
	f, _ := os.Create("cpuprof.out")
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	// enable logging
	logFile, _ := os.OpenFile("sync.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	log.SetOutput(logFile)
	log.SetFlags(log.Ltime | log.Lshortfile)

	mainCmd, opts := cmd.CreateMainCommand()
	mainCmd.AddCommand(cmd.CreateSendCommand(opts))
	mainCmd.AddCommand(cmd.CreateServerCommand())
	if err := mainCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
