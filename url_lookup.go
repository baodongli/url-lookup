package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
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

const (
	hostNameAndPort            = "host-name-and-port"
	originalPathAndQueryString = "original-path-and-query-string"
	hashTableSize              = 31
	maxUrlsCached              = 100
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

type bucket struct {
	hit      int
	lock     sync.Mutex
	urldb    URLDB
	fileName string
}

// URLHashTbl is a hash table in which each bucket contains a map of URLs
type URLHashTbl [hashTableSize]bucket

type urlLookupServer struct {
	httpPort     int
	urlCfgPath   string
	urlCachePath string
	cachedCount  int
	maxHit       int
	urlht        URLHashTbl
	lock         sync.Mutex
}

func hash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32() % hashTableSize)
}

// Save a bucket that is getting hit the least to a file, and get room for the new item
func (s *urlLookupServer) vacate(bno int, url *URL, info *URLInfo) (int, error) {
	s.lock.Lock()
	hit := s.maxHit
	bucketNo := 0
	urlCount := 0
	for i := 0; i < hashTableSize; i++ {
		if s.urlht[i].hit <= hit && len(s.urlht[i].urldb) > urlCount {
			bucketNo = i
			hit = s.urlht[i].hit
			urlCount = len(s.urlht[i].urldb)
		}
	}
	s.cachedCount = s.cachedCount - len(s.urlht[bucketNo].urldb)
	s.urlht[bucketNo].hit = 0
	s.lock.Unlock()

	log.Printf("Vacate bucket '%v' with %v urls\n", bucketNo, urlCount)
	// Vacate this bucket
	bucket := &s.urlht[bucketNo]
	bucket.lock.Lock()

	// if it's the same bucket, add it to the map first
	if url != nil && bno == bucketNo {
		bucket.urldb[*url] = info
	}

	entries := &URLs{
		URLEntries: make([]URLDBEntry, len(bucket.urldb)),
	}
	index := 0
	for url1, info := range bucket.urldb {
		log.Printf("url1: %v, info: %v", url1, *info)
		entries.URLEntries[index].HostAndPort = url1.hostAndPort
		entries.URLEntries[index].OriginalPath = url1.originalPath
		entries.URLEntries[index].Category = info.Category
		entries.URLEntries[index].Safe = info.Safe
		index++
		delete(bucket.urldb, url1)
	}

	data, err := json.Marshal(entries)
	if err != nil {
		log.Printf("failed to Marshall: %v\n", err)
		return 0, err
	}
	if err = ioutil.WriteFile(bucket.fileName, data, 0666); err != nil {
		log.Printf("failed to Marshall: %v\n", err)
		return 0, err
	}
	bucket.lock.Unlock()
	return bucketNo, nil
}

func (s *urlLookupServer) addToCache(url *URL, info *URLInfo) error {
	// Maximum cache capacity reached.
	bucketNo := hash(url.hostAndPort)
	if s.cachedCount >= maxUrlsCached {
		bno, err := s.vacate(bucketNo, url, info)
		if err != nil {
			return err
		}

		// The bucket for this item is the same as the one vacated
		// The item has been added
		if bno == bucketNo {
			return nil
		}

	}
	// Add this item
	log.Printf("add one url %v in bucket '%v'\n", url, bucketNo)
	bucket := &s.urlht[bucketNo]
	bucket.lock.Lock()
	bucket.urldb[*url] = info
	bucket.lock.Unlock()
	s.lock.Lock()
	s.cachedCount++
	s.lock.Unlock()
	return nil
}

func (s *urlLookupServer) lookupURL(request *restful.Request, response *restful.Response) {
	host := request.PathParameter(hostNameAndPort)
	original := request.PathParameter(originalPathAndQueryString)

	url := URL{hostAndPort: host, originalPath: original}
	bucketNo := hash(url.hostAndPort)
	bucket := &s.urlht[bucketNo]
	s.lock.Lock()
	bucket.hit++
	s.lock.Unlock()

	var err error
	var urlinfo *URLInfo
	log.Printf("Look up url %v in bucket '%v' with '%v' urls\n", url, bucketNo, len(bucket.urldb))
	bucket.lock.Lock()
	// Load the bucket
	if len(bucket.urldb) == 0 {
		bucket.lock.Unlock()
		err = s.loadFromFile(bucket.fileName)
		bucket.lock.Lock()
	}

	if err == nil {
		urlinfo = bucket.urldb[url]
		if urlinfo == nil {
			urlinfo = notFound
		}
	}
	bucket.lock.Unlock()
	if err != nil {
		if err := response.WriteEntity("Internal error"); err != nil {
			fmt.Printf("Failed to write entry: %v", err)
		}
	} else {
		if err := response.WriteEntity(urlinfo); err != nil {
			fmt.Printf("Failed to write entry: %v", err)
		}
	}
}

func (s *urlLookupServer) loadFromFile(path string) error {
	log.Printf("Loading from %v", path)
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

	log.Printf("Adding %v urls", len(urls.URLEntries))
	for _, urlinfo := range urls.URLEntries {
		url := URL{hostAndPort: urlinfo.HostAndPort, originalPath: urlinfo.OriginalPath}
		info := &URLInfo{Category: urlinfo.Category, Safe: urlinfo.Safe}
		err = s.addToCache(&url, info)
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
					if supportedExtensions[filepath.Ext(event.Name)] {
						s.loadFromFile(event.Name)
					}
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
		lock:         sync.Mutex{},
	}

	for i := 0; i < hashTableSize; i++ {
		ulServer.urlht[i].hit = 0
		ulServer.urlht[i].lock = sync.Mutex{}
		ulServer.urlht[i].urldb = make(URLDB)
		ulServer.urlht[i].fileName = fmt.Sprintf("%s/bucket%v.json", urlCachePath, i)
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
