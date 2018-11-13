package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	restful "github.com/emicklei/go-restful"
)

const (
	hostNameAndPort            = "host-name-and-port"
	originalPathAndQueryString = "original-path-and-query-string"
)

// URL defines a URL
type URL struct {
	hostAndPort  string
	originalPath string
}

// URLInfo defines information for a URL
type URLInfo struct {
	Category string `json:"category"`
	Safe     bool   `json:"safe"`
}

// URLDB stores URLs and their information
type URLDB map[URL]*URLInfo

var urldb URLDB

func waitSignal(stop chan struct{}) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	close(stop)
}

func lookupURL(request *restful.Request, response *restful.Response) {
	host := request.PathParameter(hostNameAndPort)
	original := request.PathParameter(originalPathAndQueryString)

	url := URL{hostAndPort: host, originalPath: original}
	urlinfo := urldb[url]
	fmt.Printf("urlinfo: %v\n", urlinfo)
	if err := response.WriteEntity(urlinfo); err != nil {
		fmt.Printf("Failed to write entry: %v", err)
	}
}

func loadURLs() {
	url1 := URL{hostAndPort: "www.terror.com:80", originalPath: "pipe-recipes"}
	url1info := &URLInfo{Category: "terrorism", Safe: false}
	url2 := URL{hostAndPort: "www.meet.com:80", originalPath: "findme"}
	url2info := &URLInfo{Category: "social", Safe: true}
	urldb[url1] = url1info
	urldb[url2] = url2info
}

func main() {
	container := restful.NewContainer()
	ws := &restful.WebService{}
	ws.Produces(restful.MIME_JSON)
	ws.Route(ws.
		GET(fmt.Sprintf("/urlinfo/1/{%s}/{%s}", hostNameAndPort, originalPathAndQueryString)).
		To(lookupURL).
		Doc("URL lookup service").
		Param(ws.PathParameter(hostNameAndPort, "Host name and port as <host>:<port>").DataType("string")).
		Param(ws.PathParameter(originalPathAndQueryString, "Original path and query string").DataType("string")))
	container.Add(ws)

	httpServer := &http.Server{
		Addr:    ":1688",
		Handler: container,
	}

	listener, err := net.Listen("tcp", ":1688")
	if err != nil {
		fmt.Printf("Listen to port 1688 failed: %v", err)
		os.Exit(1)
	}

	urldb = make(URLDB)
	loadURLs()
	go func() {
		if err = httpServer.Serve(listener); err != nil {
			fmt.Printf("Failed to serve request: %v", err)
		}
	}()

	stop := make(chan struct{})
	waitSignal(stop)
}
