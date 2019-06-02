package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/guilledipa/link"
)

const xmlns = "http://www.sitemaps.org/schemas/sitemap/0.9"

var (
	urlSite  = flag.String("url_site", "https://www.calhoun.io/", "Root URL to sitemap.")
	maxDepth = flag.Int("max_depth", 3, "Max recursion depth.")
)

// For the XML
type loc struct {
	Value string `xml:"loc"`
}
type urlset struct {
	Xmlns string `xml:"xmlns,attr"`
	Urls  []loc  `xml:"url"`
}

func parseURL(url string) ([]link.Link, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	parsedHTMLTree, err := link.ParseHTML(res.Body)
	if err != nil {
		return nil, err
	}
	nodes := link.GetLinkNodes(parsedHTMLTree)
	return link.GetLinks(nodes), nil
}

func cleanNonDomain(links []link.Link, siteMap map[string]bool) map[string]bool {
	uSite, err := url.Parse(*urlSite)
	if err != nil {
		log.Fatalf("url.Parse(%s): %v", *urlSite, err)
	}
	for _, l := range links {
		u, err := url.Parse(l.Href)
		if err != nil {
			log.Printf("url.Parse(%s): %v", l.Href, err)
			continue
		}
		if u.IsAbs() && (u.Hostname() != uSite.Hostname()) {
			log.Printf("Ignoring %s", l.Href)
			continue
		}
		if _, ok := siteMap[strings.TrimRight(u.EscapedPath(), "/")]; !ok {
			siteMap[strings.TrimRight(u.EscapedPath(), "/")] = true
		}
	}
	return siteMap
}

func scanLayer(urlStr string, layer map[string]bool) map[string]bool {
	log.Printf("scanLayer: scanning %s", urlStr)
	links, err := parseURL(urlStr)
	if err != nil {
		log.Fatalf("scanLayer: %v", err)
	}
	return cleanNonDomain(links, layer)
}

func bfs(urlStr string, maxDepth int) []string {
	uSite, err := url.Parse(urlStr)
	if err != nil {
		log.Fatalf("url.Parse(%s): %v", urlStr, err)
	}

	visited := make(map[string]bool)
	currentLayer := make(map[string]bool)
	newLayer := make(map[string]bool)

	newLayer = scanLayer(urlStr, newLayer)
	for i := 0; i <= maxDepth; i++ {
		log.Printf("bfs: parsing layer %d", i)
		currentLayer, newLayer = newLayer, make(map[string]bool)
		if len(currentLayer) == 0 {
			break
		}
		for path := range currentLayer {
			if visited[path] {
				log.Printf("bfs: path %s already scanned. Skipping.", path)
				continue
			}
			newLayer = scanLayer(fmt.Sprintf("%s://%s%s", uSite.Scheme, uSite.Host, path), newLayer)
			visited[path] = true
			for link := range newLayer {
				if _, ok := visited[link]; !ok {
					visited[link] = false
				}
			}
		}
	}
	paths := make([]string, 0, len(visited))
	for url := range visited {
		paths = append(paths, url)
	}
	return paths
}

func toXML(relPaths []string, baseURL string, Xmlns string, writer io.Writer) error {
	uSite, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("url.Parse(%s): %v", baseURL, err)
	}

	toXML := urlset{
		Xmlns: Xmlns,
	}
	for _, u := range relPaths {
		toXML.Urls = append(toXML.Urls, loc{fmt.Sprintf("%s://%s%s", uSite.Scheme, uSite.Host, u)})
	}

	enc := xml.NewEncoder(writer)
	enc.Indent("", "  ")

	return enc.Encode(toXML)
}

func main() {
	flag.Parse()

	siteURLs := bfs(*urlSite, *maxDepth)

	toXML(siteURLs, *urlSite, xmlns, os.Stdout)

}
