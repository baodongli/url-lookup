package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// URLDBEntry defines a url record
type URLDBEntry struct {
	HostAndPort  string `json:"host"`
	OriginalPath string `json:"path"`
	Category     string `json:"category"`
	Safe         bool   `json:"safe"`
}

// URLs defines a list of records
type URLs struct {
	// URLEntries contain all the url records
	URLEntries []URLDBEntry `json:"urls"`
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("Please provide a filename that contains urls")
		os.Exit(1)
	}

	urlAll, _ := ioutil.ReadFile(os.Args[1])
	// Remove the last '\n' and Split
	urls := strings.Split(string(urlAll[:len(urlAll)-1]), "\n")

	allUrls := &URLs{
		URLEntries: make([]URLDBEntry, len(urls)),
	}
	for i, url := range urls {
		parts := strings.Split(url, "/")
		allUrls.URLEntries[i].HostAndPort = parts[0] + ":80"
		allUrls.URLEntries[i].OriginalPath = parts[1]
		allUrls.URLEntries[i].Category = "bad-site"
		allUrls.URLEntries[i].Safe = false
	}

	data, _ := json.Marshal(allUrls)
	fmt.Printf("%s", data)
}
