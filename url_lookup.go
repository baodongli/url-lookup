package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"

	restful "github.com/emicklei/go-restful"
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

// URLDB stores URLs and their information
type URLDB map[URL]*URLInfo

type urlLookupServer struct {
	httpPort     int
	urlCfgPath   string
	urlCachePath string
	urldb        URLDB
}

const (
	hostNameAndPort            = "host-name-and-port"
	originalPathAndQueryString = "original-path-and-query-string"
)

var (
	ulServer            *urlLookupServer
	supportedExtensions = map[string]bool{
		".json": true,
	}
	notFound = &URLInfo{
		Category: "Unknown",
		Safe:     false,
	}
)

func (s *urlLookupServer) lookupURL(request *restful.Request, response *restful.Response) {
	host := request.PathParameter(hostNameAndPort)
	original := request.PathParameter(originalPathAndQueryString)

	url := URL{hostAndPort: host, originalPath: original}
	urlinfo := s.urldb[url]
	if urlinfo == nil {
		urlinfo = notFound
	}
	if err := response.WriteEntity(urlinfo); err != nil {
		fmt.Printf("Failed to write entry: %v", err)
	}
}

func (s *urlLookupServer) loadURLs() error {
	err := filepath.Walk(s.urlCfgPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !supportedExtensions[filepath.Ext(path)] || (info.Mode()&os.ModeType) != 0 {
			return nil
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("Failed to read %s: %v", path, err)
			return err
		}

		var urls URLs
		if e := json.Unmarshal(data, &urls); e != nil {
			fmt.Printf("Failed to unmarshal %s: %v", path, err)
		}

		for _, urlinfo := range urls.URLEntries {
			url := URL{hostAndPort: urlinfo.HostAndPort, originalPath: urlinfo.OriginalPath}
			info := &URLInfo{Category: urlinfo.Category, Safe: urlinfo.Safe}
			s.urldb[url] = info
		}
		return nil
	})
	return err
}

func newLookupServer(httpPort int, urlCfgPath, urlCachePath string, stop <-chan struct{}) error {
	ulServer = &urlLookupServer{
		httpPort:     httpPort,
		urlCfgPath:   urlCfgPath,
		urlCachePath: urlCachePath,
		urldb:        make(URLDB),
	}

	container := restful.NewContainer()
	ws := &restful.WebService{}
	ws.Produces(restful.MIME_JSON)
	ws.Route(ws.
		GET(fmt.Sprintf("/urlinfo/1/{%s}/{%s}", hostNameAndPort, originalPathAndQueryString)).
		To(ulServer.lookupURL).
		Doc("URL lookup service").
		Param(ws.PathParameter(hostNameAndPort, "Host name and port as <host>:<port>").DataType("string")).
		Param(ws.PathParameter(originalPathAndQueryString, "Original path and query string").DataType("string")))
	container.Add(ws)

	httpAddr := fmt.Sprintf(":%v", httpPort)
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: container,
	}

	listener, err := net.Listen("tcp", httpAddr)
	if err != nil {
		fmt.Printf("Listen to port %v failed", httpPort)
		return err
	}

	if err := ulServer.loadURLs(); err != nil {
		fmt.Printf("Failed to load URLs: %v", err)
	}

	go func() {
		if err = httpServer.Serve(listener); err != nil {
			fmt.Printf("Failed to serve request: %v", err)
		}
	}()
	return nil
}
