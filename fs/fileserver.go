package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	ds "github.com/benjamonnguyen/deadsimple"
	"github.com/benjamonnguyen/deadsimple/fs/middleware"
	"github.com/charmbracelet/log"
)

type dir struct {
	path    string
	name    string
	rss     bool
	sitemap bool
}

type dirslice []dir

func (ds *dirslice) String() string {
	return fmt.Sprintf("%#v", ds)
}

func (ds *dirslice) Set(s string) error {
	var d dir
	splits := strings.SplitN(s, ",", 2)
	if !path.IsAbs(splits[0]) {
		panic("use absolute path: " + splits[0])
	}
	d.path = path.Clean(splits[0]) + string(os.PathSeparator)
	d.name = strings.ReplaceAll(d.path, string(os.PathSeparator), "/")
	if len(splits) > 1 {
		if err := d.parseOpts(splits[1]); err != nil {
			return err
		}
	}

	*ds = append(*ds, d)
	return nil
}

func (d *dir) parseOpts(opts string) error {
	it := strings.SplitSeq(opts, ",")
	for s := range it {
		kv := strings.SplitN(s, ":", 2)
		key := kv[0]
		val := ""
		if len(kv) > 1 {
			val = kv[1]
		}
		switch key {
		case "name":
			d.name = val
			if !strings.HasSuffix(d.name, "/") {
				// terminate url paths with forward slash
				d.name += "/"
			}
		case "rss":
			// d.rss = true
			panic("rss not implemented")
		case "sitemap":
			d.sitemap = true
		}
	}
	return nil
}

var (
	l         ds.Logger = log.Default()
	dirs      dirslice
	addr      string
	regenFreq time.Duration
)

func main() {
	// flags
	parseFlags()

	s := http.Server{
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		IdleTimeout:       60 * time.Second,
		Addr:              addr,
	}

	// to check for uniqueness
	dirPaths := make(map[string]struct{})
	names := map[string]struct{}{}

	// serve sitemap
	names["/sitemap.xml"] = struct{}{}
	var sitemapFile *os.File
	defer sitemapFile.Close()
	http.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		if sitemapFile == nil {
			http.Error(w, "", http.StatusServiceUnavailable)
			return
		}
		stat, err := sitemapFile.Stat()
		if err != nil {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), sitemapFile)
	})
	l.Info("serving /sitemap.xml")
	// regen loop
	go func() {
		for {
			newSitemap, err := generateSitemap()
			if err != nil {
				l.Error("failed to generate sitemap", "err", err)
			} else {
				tmp := sitemapFile
				sitemapFile = newSitemap
				_ = tmp.Close()
				l.Info("generated sitemap")
			}
			time.Sleep(regenFreq)
		}
	}()

	// serve files from configured directories
	for _, dir := range dirs {
		// check for uniqueness
		if _, exists := dirPaths[dir.path]; exists {
			panic(dir.path + " already exists")
		}
		if _, exists := names[dir.name]; exists {
			panic(dir.name + " already exists")
		}
		dirPaths[dir.path] = struct{}{}
		names[dir.name] = struct{}{}

		//
		fs := http.StripPrefix(dir.name, http.FileServer(http.Dir(dir.path)))
		http.Handle(dir.name, middleware.Log(l, fs))
		l.Infof("serving %s at %s", dir.path, dir.name)
	}

	// start server
	l.Info("starting file server", "addr", addr, "paths", dirs)
	l.Fatal(s.ListenAndServe())
}

type url struct {
	XMLName xml.Name `xml:"url"`
	Loc     string   `xml:"loc"`
	Lastmod string   `xml:"lastmod"`
}

type urlset struct {
	XMLName xml.Name `xml:"urlset"`
	XMLNS   string   `xml:"xmlns,attr"`
	URLs    []url    `xml:"url"`
}

func generateSitemap() (*os.File, error) {
	var all []url
	baseURL := "http://"
	if strings.HasPrefix(addr, ":") {
		baseURL += "localhost" + addr
	} else {
		baseURL += addr
	}
	for _, dir := range dirs {
		if dir.sitemap {
			urls, err := extractURLs(dir, baseURL)
			if err != nil {
				return nil, err
			}
			all = append(all, urls...)
			l.Debugf("extracted %d urls from %s", len(urls), dir.path)
		}
	}
	if len(all) > 0 {
		// Marshal with proper header
		sitemap := urlset{
			XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
			URLs:  all,
		}
		output, err := xml.MarshalIndent(sitemap, "", "  ")
		if err != nil {
			return nil, err
		}
		xmlContent := append([]byte(`<?xml version="1.0" encoding="UTF-8"?>`+"\n"), output...)
		// Write to tmp file
		tmp, err := os.CreateTemp("", "sitemap.xml")
		if err != nil {
			return nil, err
		}
		if _, err := tmp.Write(xmlContent); err != nil {
			return nil, err
		}
		return tmp, nil
	}
	return nil, nil
}

func extractURLs(dir dir, baseURL string) ([]url, error) {
	var urls []url
	err := filepath.WalkDir(dir.path, func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fname := filepath.Base(filePath)
		if fname == "sitemap.xml" {
			return nil
		}
		// Skip hidden files (starting with '.')
		if strings.HasPrefix(fname, ".") {
			return nil
		}
		// Get file info for modification time
		info, err := d.Info()
		if err != nil {
			return err
		}
		// Get relative path from dir.path
		relPath, err := filepath.Rel(dir.path, filePath)
		if err != nil {
			return err
		}
		// Construct URL path with leading slash
		urlPath := path.Join("/", dir.name, filepath.ToSlash(relPath))
		fullURL := baseURL + urlPath
		// Format modification date as YYYY-MM-DD
		modTime := info.ModTime().Format("2006-01-02")
		urls = append(urls, url{
			Loc:     fullURL,
			Lastmod: modTime,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	//
	return urls, nil
}

func parseFlags() {
	var err error
	flag.Var(&dirs, "p", "path to directory containing files to serve and options (ex. /path/to/files,name:files,sitemap)")
	flag.StringVar(&addr, "addr", ":8080", "http server address (default: :8080)")
	rescan := flag.String("regen", "24h", "sitemap regeneration frequency (default: 24h)")
	flag.Parse()
	regenFreq, err = time.ParseDuration(*rescan)
	if err != nil {
		l.Fatal("failed to parse rescan arg", "err", err)
	}
}
