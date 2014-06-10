package main

import (
	"errors"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const BOTKEY = "OA2ydginf4cUsA"
const BOTNAME = "argusdusty"

const DEBUGMODE = 2 // -1=no debug, 0=only errors, 1=sent commands+simple debug, 2=recieved data+pings+advanced debug, 3=raw sent data

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		err := errors.New("Too few inputs")
		logFatal(err)
	}
	host := args[0]
	port := args[1]
	trackName := ""
	carCount := 1
	password := ""
	botname := BOTNAME
	botkey := BOTKEY
	if true {
		if len(args) >= 3 {
			trackName = args[2]
		}
		if len(args) >= 4 {
			carCount, _ = strconv.Atoi(args[3])
		}
		if len(args) >= 5 {
			password = args[4]
		}
		if len(args) >= 6 {
			botname = args[5]
		}
	} else {
		if len(args) >= 3 {
			botname = args[2]
		}
		if len(args) >= 4 {
			botkey = args[3]
		}
	}
	logMessage(0, "Connecting with parameters: host="+host+", port="+port+", botname="+botname+", botkey="+botkey)
	conn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		logFatal(err)
	}
	defer conn.Close()
	Bot := NewFullBot(conn)
	Bot.Run(botname, botkey, trackName, carCount, password)
}
