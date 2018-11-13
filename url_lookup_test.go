package main

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"sync"
	"testing"
)

var entries1 = &URLs{
	URLEntries: []URLDBEntry{
		URLDBEntry{
			HostAndPort:  "www.cnn.com:80",
			OriginalPath: "news",
			Category:     "news",
			Safe:         true,
		},
		URLDBEntry{
			HostAndPort:  "www.terror.com:80",
			OriginalPath: "bomb-recipes",
			Category:     "terrorism",
			Safe:         false,
		},
		URLDBEntry{
			HostAndPort:  "www.food.com:80",
			OriginalPath: "recipes",
			Category:     "food",
			Safe:         true,
		},
	},
}

var entries2 = &URLs{
	URLEntries: []URLDBEntry{
		URLDBEntry{
			HostAndPort:  "www.espn.com:80",
			OriginalPath: "programming",
			Category:     "sports",
			Safe:         true,
		},
		URLDBEntry{
			HostAndPort:  "www.fun.com:80",
			OriginalPath: "movies",
			Category:     "violence",
			Safe:         false,
		},
		URLDBEntry{
			HostAndPort:  "www.furniture.com:80",
			OriginalPath: "all-styles",
			Category:     "shopping",
			Safe:         true,
		},
		URLDBEntry{
			HostAndPort:  "www.rebellion.com:80",
			OriginalPath: "strategies",
			Category:     "violence",
			Safe:         false,
		},
	},
}

// Test URL load with two url config files
func TestLoadUrls(t *testing.T) {
	urlCfgPath, err := ioutil.TempDir("", "urlcfg")
	if err != nil {
		t.Errorf("Failed to create tmp dir: %v\n", err)
		return
	}

	data, err := json.Marshal(entries1)
	if err != nil {
		t.Errorf("Failed to Marshall: %v\n", err)
		return
	}
	tmpfn := filepath.Join(urlCfgPath, "urlcfg1.json")
	if err := ioutil.WriteFile(tmpfn, data, 0666); err != nil {
		t.Errorf("Failed to write to file %v: %v\n", tmpfn, err)
	}

	data, err = json.Marshal(entries2)
	if err != nil {
		t.Errorf("Failed to Marshall: %v\n", err)
		return
	}
	tmpfn = filepath.Join(urlCfgPath, "urlcfg2.json")
	if err := ioutil.WriteFile(tmpfn, data, 0666); err != nil {
		t.Errorf("Failed to write to file %v: %v\n", tmpfn, err)
	}
	//defer os.RemoveAll(urlCfgPath)

	// Create the server
	server := &urlLookupServer{
		httpPort:     16888,
		urlCfgPath:   urlCfgPath,
		urlCachePath: "",
		urldb:        make(URLDB),
		lock:         sync.Mutex{},
	}

	// Load the URLs
	if err := server.loadURLs(); err != nil {
		t.Errorf("Failed to load url Config: %v\n", err)
	}

	// Test entries1 match
	for _, entry := range entries1.URLEntries {
		url := URL{
			hostAndPort:  entry.HostAndPort,
			originalPath: entry.OriginalPath,
		}
		info := server.urldb[url]
		if info.Category != entry.Category || info.Safe != entry.Safe {
			t.Errorf("Test failed with unmatched record: %v\n", err)
		}
	}

	// Test entries2 match
	for _, entry := range entries2.URLEntries {
		url := URL{
			hostAndPort:  entry.HostAndPort,
			originalPath: entry.OriginalPath,
		}
		info := server.urldb[url]
		if info.Category != entry.Category || info.Safe != entry.Safe {
			t.Errorf("Test failed with unmatched record: %v\n", err)
		}
	}
}
