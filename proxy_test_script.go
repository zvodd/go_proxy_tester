package main

import (
	"bufio"
	"encoding/json"
	"errors"
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

var AUTH *proxy.Auth = &proxy.Auth{}
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
	var config Config
	authuserPtr := flag.String("user", "", "override Auth username")
	authpassPtr := flag.String("pass", "", "override Auth password")
	urlPtr := flag.String("url", "", "URL to HTTP GET check proxy connection.")
	portPtr := flag.String("port", "", "override Proxy port")
	configPtr := flag.String("config", "", "json config file")
	flag.Parse()

	if len(*configPtr) > 0 {
		jsonFile, err := os.Open(*configPtr)
		if err != nil {
			logger.Fatalf("Error opening config file \"%s\": %v\n", *configPtr, err)
		}
		defer jsonFile.Close()

		body, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(body, &config)
		if err != nil {
			logger.Fatalf("Error parsing config file \"%s\": %v\n", *configPtr, err)
		}
	}

	// Tedious flag/config overriding.
	// TODO use viper

	if len(config.User) > 0 {
		AUTH.User = config.User
	}
	if len(*authuserPtr) > 0 {
		AUTH.User = *authuserPtr
	}
	if len(config.Password) > 0 {
		AUTH.Password = config.Password
	}
	if len(*authpassPtr) > 0 {
		AUTH.Password = *authpassPtr
	}

	if len(config.PortOveride) > 0 {
		PORTOVERIDE = config.PortOveride
	}
	if len(*portPtr) > 0 {
		PORTOVERIDE = *portPtr
	}

	if len(config.TestUrl) > 0 {
		TESTURL = config.TestUrl
	}
	if len(*urlPtr) > 0 {
		TESTURL = *urlPtr
	}

	var file io.Reader
	if len(flag.Args()) < 1 {
		logger.Fatalln("No filename for proxy list supplied.")
	} else {
		var err error
		filename := flag.Args()[0]
		file, err = os.Open(filename)
		if err != nil {
			logger.Fatalf("Failed to open input file \"%s\".", filename)
		}

	}

	proxies := parseFile(file)
	proccessProxies(proxies)
}

func proccessProxies(proxies []string) {
	batchsize := CONNECTION_PARALLEL
	lenproxies := len(proxies)
	if lenproxies < batchsize {
		batchsize = lenproxies
	}
	tcount := 0
	fincount := 0
	taskchan := make(chan int)

	mytask := func(proxy string, tnum int) {
		// logger.Printf("Task %d started.", tnum)
		body, err := makeRequest(proxy, TESTURL)
		if err != nil {
			logger.Printf("Failure for '%s': %v\n\n", proxy, err)
		} else {
			fmt.Printf("Success for '%s' :\n%s\n", proxy, body)
		}
		taskchan <- tnum
	}
	// spin up initial tasks
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

func parseFile(file io.Reader) (output []string) {
	scanner := bufio.NewScanner(bufio.NewReader(file))
	scanner.Split(bufio.ScanLines)
	for i := 0; scanner.Scan(); i++ {
		address, err := parseLine(scanner.Text())
		if err != nil {
			logger.Printf("Warning, Parsing line %d: %v", i, err)
		} else {
			output = append(output, address)
		}
	}
	return output
}

func parseLine(line string) (string, error) {
	values := strings.Split(line, ",")
	if len(values) < 2 {
		return "", errors.New("Parse error. Not enough values to unpack.")
	}
	adds_str := values[1]
	addresses := strings.Split(adds_str, "|")

	if len(addresses) < 1 {
		return "", errors.New("Parse error. Invalid addresses.")
	}
	firatAdd := addresses[0]

	if len(firatAdd) < 1 {
		return "", errors.New("Parse error. Invalid address.")
	}

	pair := strings.Split(firatAdd, ":")
	if len(pair) != 2 {
		return "", errors.New("Parse error. Invalid 'address:port' format.")
	}

	host, port := pair[0], pair[1]
	if len(PORTOVERIDE) > 0 {
		port = PORTOVERIDE
	}

	return fmt.Sprintf("%s:%s", host, port), nil
}

func makeRequest(address, testUrl string) (string, error) {

	// create a socks5 dialer
	dialer, err := proxy.SOCKS5("tcp", address, AUTH, proxy.Direct)
	if err != nil {
		return "", errors.New(fmt.Sprint("Proxy setup failed:", err))
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
		return "", errors.New(fmt.Sprint("Request setup failed:", err))
	}
	// use the http client to fetch the page
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", errors.New(fmt.Sprint("Request failed:", err))
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Reponse terminated early: %v", err))
	}

	return string(b), nil
}
