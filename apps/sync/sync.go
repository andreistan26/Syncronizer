package main

import (
	"os"

	"github.com/andreistan26/sync/src/cmd"
)

func main() {
	//f, _ := os.Create("cpuprof.out")
	//pprof.StartCPUProfile(f)
	//defer pprof.StopCPUProfile()
	mainCmd, opts := cmd.CreateMainCommand()
	mainCmd.AddCommand(cmd.CreateSendCommand(opts))
	mainCmd.AddCommand(cmd.CreateServerCommand(opts))
	if err := mainCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
