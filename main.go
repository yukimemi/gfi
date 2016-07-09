package main

import (
	"os"
	"runtime"

	"github.com/Sirupsen/logrus"
)

func main() {
	// New cli
	cli := &CLI{
		log:       logrus.New(),
		semaphore: make(chan int, runtime.NumCPU()),
	}
	os.Exit(cli.Run(os.Args))
}
