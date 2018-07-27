// File opening and other miscellaneous functions
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/BurntSushi/toml"
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
	// Slice with other miscellaneous data the bot needs to remember between boots
	other_data = readfile("other_data.txt")
)

func init() {
	_, err := toml.DecodeFile("data/config.toml", &config)
	checkErr(err)
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
				pushlog()
				checkduplicates()
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