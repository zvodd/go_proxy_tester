package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type HostEntry struct {
	Name string `json:string`
	Host string `json:string`
}

var logger = log.New(os.Stderr, "", 0)
var output io.Writer = os.Stdout

func main() {
	inputzipPtr := flag.String("inzip", "", "zip file containing .opvn files")
	overwritePortPtr := flag.String("port", "", "replace port entry with number between 1 and 65535")
	useJsonPtr := flag.Bool("json", false, "export as json")
	filterPtr := flag.String("filter", "", "regex filter names")
	flag.Parse()

	if len(*overwritePortPtr) == 0 {
		overwritePortPtr = nil
	} else {
		i, err := strconv.Atoi(*overwritePortPtr)
		if err != nil || i < 1 || i > 65535 {
			logger.Println("Invalid port number specified.")
			return
		}
	}

	hs := make([]HostEntry, 0)
	allFilesInZip(*inputzipPtr, func(zf zip.File, extfile io.ReadCloser) {
		hs = append(hs, HostEntry{zf.Name, enumerateRemoteEntries(extfile, overwritePortPtr)})
	})
	// regex filter Names
	if len(*filterPtr) > 0 {
		re, err := regexp.Compile(*filterPtr)
		if err != nil {
			logger.Fatal(err)
		}
		// filter slice in place  -  https://github.com/golang/go/wiki/SliceTricks
		nb := hs[:0]
		for _, x := range hs {
			if re.Match([]byte(x.Name)) {
				nb = append(nb, x)
			}
		}
		hs = nb
	}

	if *useJsonPtr {
		b, err := json.Marshal(hs)
		if err != nil {
			logger.Fatal(err)
		}
		fmt.Print(string(b))
	} else {
		//dodgy CSV
		for _, he := range hs {
			fmt.Fprintln(output, fmt.Sprintf("%s,%s", he.Name, he.Host))
		}
	}

}

func allFilesInZip(zipfilename string, callback func(zip.File, io.ReadCloser)) {
	zipfile, err := zip.OpenReader(zipfilename)
	if err != nil {
		logger.Println("Bad zip file.")
		return
	}
	var zf *zip.File
	for _, zf = range zipfile.File {
		extfile, err := zf.Open()
		if err != nil {
			logger.Printf("Couldn't open the file \"%s\" in archive.\n", zf.Name)
			continue
		}
		callback(*zf, extfile)
	}
}

func enumerateRemoteEntries(f io.ReadCloser, overwritePortPtr *string) string {
	defer f.Close()
	addresses := make([]string, 0)
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()
		if strings.HasPrefix(line, "remote ") {
			adpair := strings.Split(strings.TrimPrefix(line, "remote "), " ")
			if len(adpair) < 2 {
				continue
			}
			host, port := adpair[0], adpair[1]
			if overwritePortPtr != nil {
				port = *overwritePortPtr
			}
			if strings.Contains(host, ".") {
				addresses = append(addresses, fmt.Sprintf("%s:%s", host, port))
			}
		}

	}
	return strings.Join(addresses, "|")
}
