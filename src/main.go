package main

import (
	"flag"
	"fmt"
	"Benchmarks"
)


var benchmark = flag.String("benchmark", "default", "choose benchmark algorithm")
var nrprocs = flag.Int("hosts", 1, "choose number of hosts.")
var port = flag.Int("port", 2000, "Choose port.")
var manager = flag.Bool("manager", true, "choose if instance is manager.")

func main() {
	flag.Parse()

	fmt.Println(*benchmark)
	switch *benchmark {

	case "barrTM":
		Benchmarks.TestBarrierTimeTM(100000, 4)
	case "locksTM":
		Benchmarks.TestLockTM(100000)
	case "SyncOpsCostTM":
		Benchmarks.TestSynchronizedReadsWritesTM(100000)
	case "NonSyncOpsCostTM":
		Benchmarks.TestNonSynchronizedReadWritesTM(100000)

	default:
		fmt.Println("Invalid input")
	}
}


