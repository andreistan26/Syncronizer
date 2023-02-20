package main

import (
	"os"
	"runtime/pprof"

	"github.com/andreistan26/sync/src/cmd"
)

func main() {
	f, _ := os.Create("cpuprof.out")
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	if err := cmd.CreateMainCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
