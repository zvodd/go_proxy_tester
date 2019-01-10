package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	ping "github.com/sparrc/go-ping"
)

var logger = log.New(os.Stderr, "", 0)

const CONNECTION_PARALLEL = 5

type HostEntry struct {
	Name string `json:string`
	Host string `json:string`
}

func main() {
	filePtr := flag.String("file", "", "file")
	flag.Parse()

	hosts, err := openAndParseFile(*filePtr)
	if err != nil {
		logger.Fatal(err)
		return
	}
	// fmt.Println(hosts)
	proccessProxies(hosts, CONNECTION_PARALLEL)
}

func openAndParseFile(filename string) ([]HostEntry, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	return parseEntries(fh)

}

func parseEntries(r io.Reader) (rv []HostEntry, err error) {
	entrybytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(entrybytes, &rv)
	return rv, err
}

// TODO Make this some sort of generic task runner.
// Starts a bunch of go routines upto $parallelMax, replenishes as routines finish.
func proccessProxies(targets []HostEntry, parallelMax int) {
	lenTargets := len(targets)
	if lenTargets < parallelMax {
		parallelMax = lenTargets
	}
	tcount := 0
	fincount := 0
	taskchan := make(chan int)

	mytask := func(target HostEntry, tnum int) {
		// logger.Printf("Task %d started.", tnum)
		hostparts := strings.Split(target.Host, ":")
		stats, err := makePing(hostparts[0], 3)
		if err != nil {
			logger.Printf("Failure \"%s\", %v\n", target, err)
		} else {
			fmt.Printf(`Success "%s", `, target.Name)
			fmt.Printf(`"%s"; `, stats.Addr)
			fmt.Printf(`%d trans, %d recv, %v%% loss; `,
				stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
			fmt.Printf("min/avg/max/stddev = %v/%v/%v/%v\n",
				stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
		}
		taskchan <- tnum
	}
	// spin up initial tasks
	for ; tcount < parallelMax; tcount++ {
		go mytask(targets[tcount], tcount)
	}

	// as tasks finish, replace them
	for fincount < lenTargets-1 {
		et := <-taskchan
		logger.Printf("Task %d finished.", et)
		fincount++
		tcount++
		// until we are out of tasks.
		if tcount < lenTargets {
			go mytask(targets[tcount], tcount)
		}
	}
	close(taskchan)
}

func makePing(target string, count int) (*ping.Statistics, error) {
	pinger, err := ping.NewPinger(target)
	if err != nil {
		return nil, err
	}
	pinger.SetPrivileged(true)
	pinger.Count = count
	pinger.Run() // blocks until finished
	stats := pinger.Statistics()
	return stats, nil
}
