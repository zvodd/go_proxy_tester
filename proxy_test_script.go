package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var logger = log.New(os.Stderr, "", 0)

var AUTH *proxy.Auth = nil
var TESTURL string = "https://httpbin.org/ip"
var PORTOVERIDE string = ""

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
		logger.Println("No file name supplied")
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

	parse_input(file)
}

func parse_input(file io.Reader) {
	scanner := bufio.NewScanner(bufio.NewReader(file))
	scanner.Split(bufio.ScanLines)

	for i := 0; scanner.Scan(); i++ {
		values := strings.Split(scanner.Text(), ",")
		if len(values) < 2 {
			logger.Println("Bad format on line:", i)
			continue
		}
		adds_str := values[1]
		addresses := strings.Split(adds_str, "|")
		if len(addresses) < 1 {
			logger.Println("Bad addresses on line,", i)
		}
		for _, remote := range addresses {
			if len(PORTOVERIDE) > 0 {
				pair := strings.Split(remote, ":")
				if len(pair) == 2 {
					remote = fmt.Sprintf("%s:%s", pair[0], PORTOVERIDE)
				}
			}
			do_request(remote, TESTURL)

		}

		if i == 200 {
			break
		}

	}
}

func do_request(address, testUrl string) bool {

	// create a socks5 dialer
	dialer, err := proxy.SOCKS5("tcp", address, AUTH, proxy.Direct)
	if err != nil {
		// logger.Println("can't connect to the proxy:", err)
		return false
	} else {
		// logger.Println("Connected to the proxy:", address)
	}
	// setup a http client
	httpTransport := &http.Transport{}
	httpClient := &http.Client{Transport: httpTransport}
	// set our socks5 as the dialer
	httpTransport.Dial = dialer.Dial
	// create a request
	req, err := http.NewRequest("GET", testUrl, nil)
	if err != nil {
		logger.Println("can't create request:", err)
		return false
	}
	// use the http client to fetch the page
	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Println("can't GET page:", err)
		return false
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading body:", err)
		return false
	}
	fmt.Println(string(b))
	return true
}
