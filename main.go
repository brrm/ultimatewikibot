// Main
package main

import (
	"os"
	"os/signal"
	"syscall"
)

func init() {
	cyclefuncs()
}

func main() {
	// Stuff to do on exit
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		pushlog()
		writefile("blacklisted_users.txt", blacklisted_users)
		os.Exit(1)
	}()
	startbot()
}
