// File opening and other miscellaneous functions
package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/tidwall/gjson"
)

type conf struct {
	BotUsername string
	Spreadsheet string
}

var (
	// Slice of blacklistedusers
	blacklisted_users = readfile("blacklisted_users.txt")
	// Config from config.toml
	config conf
	// Slice of approved subs
	approved_subs = readfile("approved_subs.txt")
	// Slice of permalinks of the bot's comments
	bot_comments []string
	// Slice of posts already replied to
	replied_posts []string
)

func init() {
	_, err := toml.DecodeFile("data/config.toml", &config)
	checkErr(err)
	updatevars()
}

// Opens given file and returns sliced string of every line inside the file
func readfile(filepath string) []string {
	// Open given file
	file, err := os.Open("data/" + filepath)
	checkErr(err)
	defer file.Close()

	// Store them in a slice
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

// Error handler
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Execute certain funcs every minute
func cyclefuncs() {
	ticker := time.NewTicker(1 * time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				updatevars()
				pushlog()
				checkscore()
				writefile("blacklisted_users.txt", blacklisted_users)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

// Writes to a specific file
func writefile(filepath string, data []string) {
	// Create file
	file, err := os.Create("data/" + filepath)
	checkErr(err)
	defer file.Close()

	// Write to it
	w := bufio.NewWriter(file)
	for _, d := range data {
		fmt.Fprintln(w, d)
	}
	err = w.Flush()
	checkErr(err)
}

// Updates bot_comments and replied_posts
func updatevars() {
	url := "https://api.reddit.com/user/" + config.BotUsername + "/comments"
	// Send GET request
	redditClient := http.Client{
		Timeout: time.Second * 2,
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	checkErr(err)
	req.Header.Set("User-Agent", "ubuntu:github.com/brrm/ultimatewikibot:v0.1 (by /u/litllsnek)")
	// Parse response
	res, err := redditClient.Do(req)
	checkErr(err)
	body, err := ioutil.ReadAll(res.Body)
	checkErr(err)
	datajson := string(body)
	// Reset vars
	bot_comments = []string{}
	replied_posts = []string{}
	// Update vars
	for i := 0; i < int(gjson.Get(datajson, "data.children.#").Int()); i++ {
		bot_comments = append(bot_comments, gjson.Get(datajson, "data.children."+strconv.Itoa(i)+".data.permalink").String())
		replied_posts = append(replied_posts, gjson.Get(datajson, "data.children."+strconv.Itoa(i)+".data.parent_id").String())
	}
}
