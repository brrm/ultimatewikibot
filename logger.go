// Logger
package main

import (
	"io/ioutil"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"gopkg.in/Iwark/spreadsheet.v2"
)

func init() {
	initsheets()
}

var logslice = [][]string{}

type logdata struct {
	Subreddit string
	Author    string
	Posttype  string
	Permalink string
	Reqtype   int
	Wikidatas []wikidata
}

func logger(ld logdata) {
	var logs []string
	logs = append(logs, time.Now().Format("01/02/2006"))
	logs = append(logs, ld.Subreddit)
	logs = append(logs, ld.Author)
	logs = append(logs, ld.Posttype)
	logs = append(logs, "https://reddit.com"+ld.Permalink)
	if ld.Reqtype == 1 {
		logs = append(logs, "wikibot what is?")
	} else {
		logs = append(logs, "URL")
	}
	var titles []string
	var wikiurls []string
	for _, wikidata := range ld.Wikidatas {
		titles = append(titles, wikidata.Title)
		wikiurls = append(wikiurls, "https://en.wikipedia.org/wiki/"+wikidata.Canonical)
	}
	logs = append(logs, strings.Join(titles, ", "))
	logs = append(logs, strings.Join(wikiurls, ", "))

	logslice = append(logslice, logs)
}

// Pushes the logs to the spreadsheet
func pushlog() {
	// Make sure there actually is a logslice to push
	if len(logslice) > 0 {
		// Setup authentication
		data, err := ioutil.ReadFile("data/client_secret.json")
		checkErr(err)
		conf, err := google.JWTConfigFromJSON(data, spreadsheet.Scope)
		checkErr(err)
		client := conf.Client(context.TODO())

		// Access first sheet in given spreadsheet
		service := spreadsheet.NewServiceWithClient(client)
		spreadsheet, err := service.FetchSpreadsheet(config.Spreadsheet)
		checkErr(err)
		sheet, err := spreadsheet.SheetByIndex(0)
		checkErr(err)

		// Logger
		// Find next available row
		rc := len(sheet.Rows)
		for i, logs := range logslice {
			// Write the log to the row
			for j, log := range logs {
				sheet.Update(rc+i, j, log)
			}
		}
		// Push the changes
		err = sheet.Synchronize()
		checkErr(err)

		// Erase logslice
		logslice = [][]string{}
	}
}

// Sets headers for the spreadsheet if it detects that it is blank
func initsheets() {
	// Setup authentication
	data, err := ioutil.ReadFile("data/client_secret.json")
	checkErr(err)
	conf, err := google.JWTConfigFromJSON(data, spreadsheet.Scope)
	checkErr(err)
	client := conf.Client(context.TODO())
	// Access first sheet in given spreadsheet
	service := spreadsheet.NewServiceWithClient(client)
	spreadsheet, err := service.FetchSpreadsheet(config.Spreadsheet)
	checkErr(err)
	sheet, err := spreadsheet.SheetByIndex(0)
	checkErr(err)
	// Writes headers
	headers := []string{"Time", "Subreddit", "Author", "Post Type", "Permalink", "Request Type", "Wikipedia Pages", "Wikipedia URLs"}
	for i, header := range headers {
		sheet.Update(0, i, header)
	}
	err = sheet.Synchronize()
	checkErr(err)
}
