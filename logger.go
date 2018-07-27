// Logger
package main

import (
	"io/ioutil"
	"strings"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"gopkg.in/Iwark/spreadsheet.v2"
	"github.com/tidwall/gjson"
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

// Checks for duplicate replies the bot may have made and deletes them
func checkduplicates() {
	// Get permalinks from sheet
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
	// Get all permalinks from the permalink column
	var permalinks []string
	lastread, _ := strconv.Atoi(other_data[0])
	for i:=lastread; i<len(sheet.Rows); i++ {
		if sheet.Columns[4][i].Value != "Permalink" {
			permalinks = append(permalinks, sheet.Columns[4][i].Value)
		}
	}
	// Find duplicate permalinks
	// Create a map with each permalink and a bool (where if true the permalink is a duplicate)
	permalinkmap := make(map[string]bool)
	// Function for checking if a given string is already in a given map
	isalreadyinmap := func(item string, m map[string]bool) bool {
		for mapelement, _ := range m {
			if item == mapelement {
				return true
			}
		}
		return false
	}
	// Range through all the given permalinks
	for _, permalink := range permalinks {
		// Check if they are already in the map
		if isalreadyinmap(permalink, permalinkmap) {
			// Mark the permalink as duplicate
			permalinkmap[permalink] = true
		} else {
			// Add the permalink to the map but mark it as not a duplicate
			permalinkmap[permalink] = false
		}
	}
	// Empty slice of all the duplicates
	var duplicates []string
	// Range through each permalink in the map
	for permalink, _ := range permalinkmap {
		// If the permalink has been marked as a duplicate
		if permalinkmap[permalink] == true {
			// Add it to the duplicate slice
			duplicates = append(duplicates, permalink)
		}
	}
	// Find the actual bot replies to the permalinks and send all of them (except the first reply) to be deleted
	var deletereplies []string
	for _, duplicate := range duplicates {
		// Set URL
		url := "https://api.reddit.com"+strings.Split(duplicate, "https://reddit.com")[1]+".json"
		// Send GET request
		redditClient := http.Client{
			Timeout: time.Second * 4,
		}
		req, err := http.NewRequest(http.MethodGet, url, nil)
		checkErr(err)
		req.Header.Set("User-Agent", "ubuntu:github.com/brrm/ultimatewikibot:v0.1 (by /u/litllsnek)")
		// Parse response
		res, err := redditClient.Do(req)
		checkErr(err)
		body, err := ioutil.ReadAll(res.Body)
		checkErr(err)
		bob := string(body)
		authors := gjson.Get(bob, "1.data.children.0.data.replies.data.children.#.data.author").Array()
		// Gather all the duplicate replies
		var duplicatereplies []string
		for i, author := range authors {
			if author.String() == config.BotUsername {
				duplicatereplies = append(duplicatereplies, gjson.Get(bob, "1.data.children.0.data.replies.data.children."+strconv.Itoa(i)+".data.permalink").String())
			}
		}
		// Append all of them except the first to deletereplies slice
		for i:=1; i<len(duplicatereplies); i++ {
			deletereplies=append(deletereplies, duplicatereplies[i])
		}
	}
	// Delete duplicate replies
	for _, deletereply := range deletereplies {
		botglobal.bot.Delete("t1_" + deletereply[len(deletereply)-8:len(deletereply)-1])
	}
	// Overwrite other_data[0] so the bot knows which row of the sheet it last checked for duplicates
	other_data[0] = strconv.Itoa(len(sheet.Rows))
	writefile("other_data.txt", other_data)
}