package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	restful "github.com/emicklei/go-restful"
	"github.com/fsnotify/fsnotify"
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
	lock         sync.Mutex
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
	s.lock.Lock()
	urlinfo := s.urldb[url]
	s.lock.Unlock()
	if urlinfo == nil {
		urlinfo = notFound
	}
	if err := response.WriteEntity(urlinfo); err != nil {
		fmt.Printf("Failed to write entry: %v", err)
	}
}

func (s *urlLookupServer) loadFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("Failed to read %s: %v", path, err)
		return err
	}

	var urls URLs
	if err = json.Unmarshal(data, &urls); err != nil {
		log.Printf("Failed to unmarshal %s: %v", path, err)
		return err
	}

	for _, urlinfo := range urls.URLEntries {
		url := URL{hostAndPort: urlinfo.HostAndPort, originalPath: urlinfo.OriginalPath}
		info := &URLInfo{Category: urlinfo.Category, Safe: urlinfo.Safe}
		s.lock.Lock()
		s.urldb[url] = info
		s.lock.Unlock()
	}
	return nil
}

func (s *urlLookupServer) loadURLs() error {
	err := filepath.Walk(s.urlCfgPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !supportedExtensions[filepath.Ext(path)] || (info.Mode()&os.ModeType) != 0 {
			return nil
		}

		err = s.loadFromFile(path)
		return err
	})
	return err
}

func (s *urlLookupServer) watchForUpdate() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				// In case of change, load the changed/added file
				// Only support adding new files and new entries for now
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					s.loadFromFile(event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	// Watch the URL configuration path
	err = watcher.Add(s.urlCfgPath)
	if err != nil {
		return err
	}
	return nil
}

func newLookupServer(httpPort int, urlCfgPath, urlCachePath string, stop <-chan struct{}) error {
	ulServer = &urlLookupServer{
		httpPort:     httpPort,
		urlCfgPath:   urlCfgPath,
		urlCachePath: urlCachePath,
		urldb:        make(URLDB),
		lock:         sync.Mutex{},
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

	// Create the listener for the web server
	listener, err := net.Listen("tcp", httpAddr)
	if err != nil {
		log.Printf("Listen to port %v failed", httpPort)
		return err
	}

	log.Println("Loading URLs...")
	// Load URLs from configuration files
	if err := ulServer.loadURLs(); err != nil {
		log.Printf("Failed to load URLs: %v", err)
	}

	log.Println("Starting to watch for update...")
	if err := ulServer.watchForUpdate(); err != nil {
	}

	// Start a go routine to serve http requests
	go func() {
		log.Println("Start serving ...")
		if err = httpServer.Serve(listener); err != nil {
			log.Printf("Failed to serve request: %v", err)
		}
	}()
	return nil
}
