// Bot and all its handlers
package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/brrm/graw"
	"github.com/brrm/graw/reddit"
	"github.com/tidwall/gjson"
)

// Allows the reddit.Bot to be accessed anywhere
var botglobal ultimatewikibot

type ultimatewikibot struct {
	bot reddit.Bot
}

// Starts the bot
func startbot() {
	// Create new bot from .agent
	if bot, err := reddit.NewBotFromAgentFile("data/bot.agent", 2*time.Second); err != nil {
		log.Fatal("Failed to create bot handle: ", err)
	} else {
		// Allow the bot to be accessed anywhere
		botglobal = ultimatewikibot{bot: bot}
		// Create cfg from approved_subs.txt
		cfg := graw.Config{Subreddits: approved_subs, SubredditComments: approved_subs, Messages: true}
		handler := &ultimatewikibot{bot: bot}
		if _, wait, err := graw.Run(handler, bot, cfg); err != nil {
			log.Fatal("Failed to start graw run: ", err)
		} else {
			log.Fatal("graw run failed: ", wait())
		}
	}
}

// PM handler for blacklisting
func (r *ultimatewikibot) Message(m *reddit.Message) error {
	if m.Subject == "Blacklist" && m.Body == "Me" {
		blacklisted_users = append(blacklisted_users, m.Author)
	}
	return nil
}

// Post handler
func (r *ultimatewikibot) Post(p *reddit.Post) error {
	// Checks if it has replied to the post
	if didnotreply(p.Name) {
		// Check if post is link or text
		var content string
		var posttype string
		if p.SelfText == "" {
			content = p.URL
			posttype = "Link"
		} else {
			content = p.SelfText
			posttype = "Text"
		}
		// If it's the bot's own post, don't reply to it
		if p.Author == config.BotUsername {
			return nil
		}
		// Check if author is blacklisted
		if validateauthor(p.Author) {
			// Get queries
			reqtype, queries := getqueries(content)
			// If queries found
			if reqtype > 0 {
				// Generate wikidata and validate queries
				valid, wds := validatequeries(queries)
				if valid {
					// Log all the data
					logger(logdata{Subreddit: p.Subreddit, Author: p.Author, Posttype: posttype, Permalink: p.Permalink, Reqtype: reqtype, Wikidatas: wds})
					// Add the post permalink to replied_posts
					replied_posts = append(replied_posts, p.Permalink)
					// Generate reply out of wikidata
					return r.bot.Reply(p.Name, formatreply(wds, reqtype))
				}
				return nil
			}
			return nil
		}
		return nil
	}
	return nil
}

// Comment handler
func (r *ultimatewikibot) Comment(c *reddit.Comment) error {
	// Checks if it has replied to the comment
	if didnotreply(c.Name) {
		// If it's the bot's own comment don't reply to it, add the permalink to bot_comments
		if c.Author == config.BotUsername {
			bot_comments = append(bot_comments, c.Permalink)
			return nil
		}
		// Check if author is blacklisted
		if validateauthor(c.Author) {
			// Get queries
			reqtype, queries := getqueries(c.Body)
			// If queries found
			if reqtype > 0 {
				// Generate wikidata and validate queries
				valid, wds := validatequeries(queries)
				if valid {
					// Log all the data
					logger(logdata{Subreddit: c.Subreddit, Author: c.Author, Posttype: "Comment", Permalink: c.Permalink, Reqtype: reqtype, Wikidatas: wds})
					// Add the comment permalink to replied_posts
					replied_posts = append(replied_posts, c.Permalink)
					// Generate reply out of wikidata
					return r.bot.Reply(c.Name, formatreply(wds, reqtype))
				}
				return nil
			}
			return nil
		}
		return nil
	}
	return nil
}

// Returns queries as sliced string for a given string (sourced from post or comments). Also returns reqtype int. 1 - "wikibot what is", 2 - wikipedia link, 0 - none of the above
func getqueries(s string) (int, []string) {
	// Wikibot what is:
	// Convert string to lowercase
	sl := strings.ToLower(s)
	// Capatilizes first letter after every white space (unlike strings.Title which capatilizes after periods and other symbols). Also replaces whitespaces with underscores.
	title := func(s string) string {
		// Split the string into words
		words := strings.Fields(s)
		var chars []string
		// Range through words
		for i, word := range words {
			// Range through characters
			for j, r := range word {
				// If first character, capitalize it
				if j == 0 {
					chars = append(chars, string(unicode.ToUpper(r)))
				} else {
					chars = append(chars, string(r))
				}
			}
			// If not the last word, add an underscore after it
			if i < len(words)-1 {
				chars = append(chars, "_")
			}
		}
		// Join the characters back together
		return strings.Join(chars, "")
	}
	// Checks if contains "wikibot what is"
	if strings.Contains(sl, "wikibot what is") {
		// Get everything in betwen "wikibot what is" and "?", trim leading and trailing whitespaces, and capitalize words.
		re := regexp.MustCompile(`wikibot what is(.+)?\?`)
		s = title(strings.TrimSpace(re.FindStringSubmatch(sl)[1]))
		// Returns true and the string as a slice, now in query form
		return 1, []string{s}
	}

	// Wikipedia link:
	// Get all .org URLs from string (using xurls.go)
	urls := FindURL().FindAllStringSubmatch(s, -1)
	// Checks if given url is a wikipedia wiki and return the bit after /wiki/. Returns ok bool and query string.
	makequery := func(url string) (bool, string) {
		// Supported wikipedia urls (no non-english urls)
		wikiurls := []string{"en.wikipedia.org/wiki/", "www.wikipedia.org/wiki/"}
		// Return nothing if a section was linked (sections are currently not supported).
		if strings.Contains(url, "#") {
			return false, ""
		}
		if strings.Contains(url, wikiurls[0]) {
			return true, strings.TrimSpace(strings.Split(url, wikiurls[0])[1])
		}
		if strings.Contains(url, wikiurls[1]) {
			return true, strings.TrimSpace(strings.Split(url, wikiurls[1])[1])
		}
		return false, ""
	}
	// Checks if it found at least 1 URL
	if len(urls) > 0 {
		// Slice of wikipedia queries
		var queries []string
		// Range through urls
		for _, url := range urls {
			// Convert each query into a wikipedia url
			ok, query := makequery(url[0])
			if ok {
				// Check if query is already included in slice
				isnewquery := func(q string) bool {
					for _, query := range queries {
						if q == query {
							return false
						}
					}
					return true
				}
				// If the query is a wikipedia url, and not already include in the slice, add it to the slice
				if isnewquery(query) {
					queries = append(queries, query)
				}
			}
		}
		// Checks if it found at least 1 wikipedia query
		if len(queries) > 0 {
			// Return the queries
			return 2, queries
		}
		// Didn't find any wikipedia queries, return false and empty string
		return 0, []string{""}
	}

	// The comment/post did not contain any wikipedia links or did not include "wikibot what is", return false and empty string
	return 0, []string{""}
}

// Given a slice of wikidata, and type of request, returns a formatted reply string
func formatreply(wds []wikidata, reqtype int) string {
	var replysections []string
	// Header
	if reqtype == 1 {
		replysections = append(replysections, "Looks like you summoned me by using my call command \"wikibot what is?\"...\n\n")
	} else {
		if len(wds) > 1 {
			replysections = append(replysections, "Looks like you posted some wikipedia articles, let me summarize them for you...\n\n")
		} else {
			replysections = append(replysections, "Looks like you posted a wikipedia article, let me summarize it for you...\n\n")
		}
	}
	replysections = append(replysections, "Click [here](https://reddit.com/message/compose?to="+config.BotUsername+"&subject=Blacklist&message=Me) if you'd like me to stop bugging you.\n*****\n")
	// Body
	for _, wd := range wds {
		if wd.Image != "" {
			replysections = append(replysections, "**["+wd.Title+"](https://en.wikipedia.org/wiki/"+wd.Canonical+")**\n>"+wd.Extract+"\n\n[Image]("+wd.Image+")\n*****\n")
		} else {
			replysections = append(replysections, "**["+wd.Title+"](https://en.wikipedia.org/wiki/"+wd.Canonical+")**\n>"+wd.Extract+"\n\n*****\n")
		}
	}
	// Footer
	replysections = append(replysections, "**^([)** ^([About](https://np.reddit.com/r/ultimatewikibot/wiki/index)) **^(|)** ^([Source code](https://github.com/brrm/ultimatewikibot)) **^(|)** ^(Downvote to remove) **^(])**\n")

	// Return reply
	return strings.Join(replysections, "")
}

// Checks if given user is blacklisted
func validateauthor(author string) bool {
	// Check if author appears in blacklisted user slice
	for _, user := range blacklisted_users {
		if author == user {
			return false
		}
	}
	return true
}

// Validates queries and returns wikidata for them
func validatequeries(queries []string) (bool, []wikidata) {
	ok := false
	var wds []wikidata
	// Range through queries
	for _, query := range queries {
		// Get wikidata for query
		wd := getwikidata(query)
		// Check if wikidata is good
		if wikifilter(wd) {
			// Change ok to true (since it found at least 1 good query)
			ok = true
			// Add the wikidata to the return slice
			wds = append(wds, wd)
		}
	}
	return ok, wds
}

// Checks if score is below -1, if so removes comment
func checkscore() {
	// Gets the score for a given comment permalink
	getscore := func(permalink string) int64 {
		url := "https://api.reddit.com" + permalink + ".json"
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
		commentjson := string(body)
		// Get score from json
		return gjson.Get(commentjson, "1.data.children.0.data.score").Int()
	}
	// Checks scores for all of the bot's comments
	for _, bot_comment := range bot_comments {
		score := getscore(bot_comment)
		// If the score is less than -1
		if score < -1 {
			// Delete the comment (bit inside is converting permalink to id)
			botglobal.bot.Delete("t1_" + bot_comment[len(bot_comment)-8:len(bot_comment)-1])
		}
	}
}

// Checks if it the post has already been replied to
func didnotreply(name string) bool {
	for _, rp := range replied_posts {
		if name == rp {
			return false
		}
	}
	return true
}
