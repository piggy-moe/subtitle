package util

import (
	"log"
	"os"
	"runtime/pprof"

	"github.com/urfave/cli/v2"
)

var cpuProfile, memProfile *os.File

func StartProfile(ctx *cli.Context) error {
	var err error
	if ctx.IsSet("cpuprof") {
		filename := ctx.String("cpuprof")
		if cpuProfile, err = os.Create(filename); err != nil {
			log.Fatal(err)
		}
		if err := pprof.StartCPUProfile(cpuProfile); err != nil {
			log.Fatal(err)
		}
	}
	if ctx.IsSet("memprof") {
		filename := ctx.String("memprof")
		if memProfile, err = os.Create(filename); err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func StopProfile(ctx *cli.Context) error {
	if cpuProfile != nil {
		pprof.StopCPUProfile()
		cpuProfile.Close()
	}
	if memProfile != nil {
		if err := pprof.WriteHeapProfile(memProfile); err != nil {
			log.Fatal(err)
		}
		memProfile.Close()
	}
	return nil
}
