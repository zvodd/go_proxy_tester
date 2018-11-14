package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

var logger = log.New(os.Stderr, "", 0)

var AUTH *proxy.Auth = nil
var TESTURL string = "https://httpbin.org/ip"
var PORTOVERIDE string = ""

var CONNECTION_TIMEOUT = 8
var CONNECTION_PARALLEL = 3

type Config struct {
	User        string `json:"user"`
	Password    string `json:"password"`
	TestUrl     string `json:"testUrl"`
	PortOveride string `json:"portOveride"`
}

// file format : lines of ([^,]*)($VALIDHOSTNAME:$PORT)
func main() {
	configPtr := flag.String("config", "", "json config file")
	flag.Parse()
	if len(*configPtr) > 0 {
		jsonFile, err := os.Open(*configPtr)
		if err != nil {
			logger.Println("Failed to open config file.")
			os.Exit(1)
		}
		defer jsonFile.Close()
		var config Config
		body, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(body, &config)
		if err != nil {
			fmt.Printf("err was %v", err)
		}
		if len(config.User) > 0 || len(config.Password) > 0 {
			AUTH = &proxy.Auth{User: config.User, Password: config.Password}
		}
		if len(config.PortOveride) > 0 {
			PORTOVERIDE = config.PortOveride
		}
		if len(config.TestUrl) > 0 {
			TESTURL = config.TestUrl
		}
	}

	var file io.Reader
	if len(flag.Args()) < 1 {
		logger.Println("No input file name supplied.")
		// file = os.Stdin
		os.Exit(1)
	} else {
		var err error
		file, err = os.Open(flag.Args()[0])
		if err != nil {
			logger.Println("Failed to open input file.")
			os.Exit(1)
		}

	}

	proxies := parse_input(file)
	// logger.Print(proxies)
	proccessProxies(proxies)
}

func proccessProxies(proxies []string) {
	batchsize := CONNECTION_PARALLEL
	lenproxies := len(proxies)
	if lenproxies < batchsize {
		batchsize = lenproxies
	}
	//do_request(remote, TESTURL)
	tcount := 0
	fincount := 0
	taskchan := make(chan int)

	mytask := func(proxy string, tnum int) {
		// logger.Printf("Task %d started.", tnum)
		do_request(proxy, TESTURL)
		taskchan <- tnum
	}
	// spinup initial tasks
	for ; tcount < batchsize; tcount++ {
		go mytask(proxies[tcount], tcount)
	}

	// as tasks finish, replace them
	for fincount < lenproxies-1 {
		et := <-taskchan
		logger.Printf("Task %d finished.", et)
		fincount++
		tcount++
		// until we are out of tasks.
		if tcount < lenproxies {
			go mytask(proxies[tcount], tcount)
		}
	}
	close(taskchan)
}

func parse_input(file io.Reader) (output []string) {
	scanner := bufio.NewScanner(bufio.NewReader(file))
	scanner.Split(bufio.ScanLines)

	for i := 0; scanner.Scan(); i++ {
		values := strings.Split(scanner.Text(), ",")
		if len(values) < 2 {
			logger.Printf("Warning, parsing line %d.\n", i)
			continue
		}
		adds_str := values[1]
		addresses := strings.Split(adds_str, "|")
		if len(addresses) < 1 {
			logger.Printf("Warning, invalid addresses on line %d.\n", i)
		}
		for _, remote := range addresses {
			if len(PORTOVERIDE) > 0 {
				pair := strings.Split(remote, ":")
				if len(pair) == 2 {
					remote = fmt.Sprintf("%s:%s", pair[0], PORTOVERIDE)
					output = append(output, remote)
				}
			}
		}
	}
	return output
}

func do_request(address, testUrl string) bool {

	// create a socks5 dialer
	dialer, err := proxy.SOCKS5("tcp", address, AUTH, proxy.Direct)
	if err != nil {
		logger.Println("Error, proxy setup:", err)
		return false
	}
	// setup a http client
	httpTransport := &http.Transport{}
	timeout := time.Duration(CONNECTION_TIMEOUT) * time.Second
	httpClient := &http.Client{Transport: httpTransport, Timeout: timeout}
	// set our socks5 as the dialer
	httpTransport.Dial = dialer.Dial
	// create a request
	req, err := http.NewRequest("GET", testUrl, nil)
	if err != nil {
		logger.Println("Error, request setup:", err)
		return false
	}
	// use the http client to fetch the page
	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Println("Error, request failed:", err)
		return false
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error, reponse terminated:", err)
		return false
	}
	fmt.Println(string(b))
	return true
}
