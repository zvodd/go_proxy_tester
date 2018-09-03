package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

var logger = log.New(os.Stderr, "", 0)
var output io.Writer = os.Stdout

func main() {
	zread, err := zip.OpenReader("list.zip")
	if err != nil {
		logger.Println("File Fuckery.")
		return
	}
	var f *zip.File
	for _, f = range zread.File {

		fr, err := f.Open()
		if err != nil {
			logger.Println("Bad file in archive")
			continue
		}
		adr_line := strings.Join(parse_ovpn_file(fr), "|")
		fmt.Fprintln(output, fmt.Sprintf("%s,%s", f.Name, adr_line))
		// break
	}

}

func parse_ovpn_file(f io.ReadCloser) []string {
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
			ip, port := adpair[0], adpair[1]
			if strings.Contains(ip, ".") {
				addresses = append(addresses, fmt.Sprintf("%s:%s", ip, port))
			}
		}

	}
	return addresses
}
