package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/guilledipa/link"
)

var (
	url = flag.String("url", "https://www.calhoun.io/", "Root URL to sitemap.")
)

func main() {
	res, err := http.Get(*url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	parsedHTMLTree, err := link.ParseHTML(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	nodes := link.GetLinkNodes(parsedHTMLTree)
	links := link.GetLinks(nodes)
	for _, l := range links {
		fmt.Println(l)
	}

}
