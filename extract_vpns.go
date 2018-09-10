package main

import (
	"archive/zip"
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

var logger = log.New(os.Stderr, "", 0)
var output io.Writer = os.Stdout

func main() {
	inputzipPtr := flag.String("inzip", "", "zip file containing .opvn files")
	overwritePortPtr := flag.String("port", "", "replace port entry with number between 1 and 65535")
	flag.Parse()

	if len(*overwritePortPtr) == 0 {
		overwritePortPtr = nil
	} else {
		i, err := strconv.Atoi(*overwritePortPtr)
		if err != nil || i < 0 || i > 65535 {
			logger.Println("Invalid port number specified.")
			return
		}
	}

	if len(*inputzipPtr) > 0 {
		zipfile, err := zip.OpenReader(*inputzipPtr)
		if err != nil {
			logger.Println("Bad zip file.")
			return
		}
		var zf *zip.File
		for _, zf = range zipfile.File {
			fr, err := zf.Open()
			if err != nil {
				logger.Printf("Couldn't open the file \"%s\" in archive.\n", zf.Name)
				continue
			}
			fmt.Fprintln(output, fmt.Sprintf("%s,%s", zf.Name, enumerateRemoteEntries(fr, overwritePortPtr)))
		}
	}

}

func enumerateRemoteEntries(f io.ReadCloser, overwritePortPtr *string) string {
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
			if overwritePortPtr != nil {
				port = *overwritePortPtr
			}
			if strings.Contains(ip, ".") {
				addresses = append(addresses, fmt.Sprintf("%s:%s", ip, port))
			}
		}

	}
	return strings.Join(addresses, "|")
}
