// Get data from wikipedia
package main

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// Data extracted from api
type wikidata struct {
	// Title as visible on page (ex: Stack Overflow)
	Title string
	// Title as found in URL (ex: Stack_Overflow)
	Canonical string
	// Summary (ex: Stack Overflow is a privately held website...)
	Extract string
	// Main image of wiki page (if the page has no image, string will be empty)
	Image string
}

// Gets extract and image url from given query. Returns wikidata{}
func getwikidata(query string) wikidata {

	url := "https://en.wikipedia.org/api/rest_v1/page/summary/" + query

	// Get wiki page
	wikiClient := http.Client{
		Timeout: time.Second * 2,
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	checkErr(err)
	req.Header.Set("User-Agent", "ultimatewikibot")

	// Parse response
	res, err := wikiClient.Do(req)
	checkErr(err)
	body, err := ioutil.ReadAll(res.Body)
	checkErr(err)
	wikijson := string(body)

	// Get the data from json
	wiki := wikidata{Title: gjson.Get(wikijson, "title").String(), Canonical: gjson.Get(wikijson, "titles.canonical").String(), Extract: gjson.Get(wikijson, "extract").String(), Image: gjson.Get(wikijson, "originalimage.source").String()}

	return wiki
}

// Checks if the given wikipedia page is good, returns "valid" bool
func wikifilter(wiki wikidata) bool {
	// Slice of bad wiki titles
	badtitles := []string{"Not found.", "List of", "Category:", "File:", "Main Page"}
	// See if given title from wikidata matches any of the bad titles
	for _, badtitle := range badtitles {
		if strings.Contains(wiki.Title, badtitle) {
			return false
		}
	}
	// Detect if the page is a redirect
	if strings.Contains(wiki.Extract, "may refer to") {
		return false
	}
	// Return true if all is good
	return true
}
