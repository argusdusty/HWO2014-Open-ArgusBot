package main

import (
	"bufio"
	"encoding/json"
)

type Join struct {
	Name string
	Key  string
}

type Track struct {
	ID     string
	Name   string
	Pieces []TPiece
	Lanes  []float64
}

type CarID struct {
	Name  string
	Color string
}

type Car struct {
	ID                CarID
	Length            float64
	Width             float64
	GuideFlagPosition float64
}

type Race struct {
	RaceTrack    Track
	Cars         []Car
	Laps         int
	MaxLapTimeMs int
	QuickRace    bool
	Qualifying   bool
	Duration     int
}

type CarPosition struct {
	ID              CarID
	Angle           float64
	PieceIndex      int
	InPieceDistance float64
	StartLaneIndex  int
	EndLaneIndex    int
	Lap             int
}

type LapData struct {
	Lap    int
	Ticks  int
	Millis int
}

type CarResult struct {
	Car    CarID
	Result LapData
}

type GameEnd struct {
	Results  []CarResult
	BestLaps []CarResult
}

type FinishedLap struct {
	Car        CarID
	LapTime    LapData
	RaceTime   LapData
	Overall    int
	FastestLap int
}

type DNF struct {
	Car    CarID
	Reason string
}

func readMsg(reader *bufio.Reader) (msgType string, data interface{}, tick int) {
	line, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	var msg map[string]interface{}
	err = json.Unmarshal([]byte(line), &msg)
	if err != nil {
		panic(err)
	}
	msgType = msg["msgType"].(string)
	data, ok := msg["data"]
	if !ok {
		data = nil
	}
	tick = -1
	if t, ok := msg["gameTick"]; ok {
		tick = int(t.(float64))
	}
	if msgType == "carPositions" {
		logMessage(2, "Recieved message:", msgType, data, tick)
	} else {
		logMessage(1, "Recieved message:", msgType, data, tick)
	}
	return
}

func writeMsg(writer *bufio.Writer, msgtype string, data interface{}) {
	m := make(map[string]interface{})
	m["msgType"] = msgtype
	m["data"] = data
	logMessage(3, "Sending data:", m)
	var payload []byte
	payload, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	payload = append(payload, '\n')
	_, err = writer.Write(payload)
	if err != nil {
		panic(err)
	}
	writer.Flush()
}

func writeMsgTick(writer *bufio.Writer, msgtype string, data interface{}, gameTick int) {
	m := make(map[string]interface{})
	m["msgType"] = msgtype
	m["data"] = data
	m["gameTick"] = gameTick
	logMessage(3, "Sending data:", m)
	var payload []byte
	payload, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	payload = append(payload, '\n')
	_, err = writer.Write(payload)
	if err != nil {
		panic(err)
	}
	writer.Flush()
}

func sendPing(writer *bufio.Writer) {
	writeMsg(writer, "ping", nil)
	logMessage(2, "Sent ping")
}

func sendPingTick(writer *bufio.Writer, gameTick int) {
	writeMsgTick(writer, "ping", nil, gameTick)
	logMessage(2, "Sent pingTick", gameTick)
}

func sendJoin(writer *bufio.Writer, name, key string) {
	data := map[string]string{"name": name, "key": key}
	writeMsg(writer, "join", data)
	logMessage(1, "Sent join:", name, key)
}

func sendJoinRace(writer *bufio.Writer, name, key, trackName string, carCount int, password string) {
	data := map[string]interface{}{"botId": map[string]string{"name": name, "key": key}, "carCount": carCount}
	if trackName != "" {
		data["trackName"] = trackName
	}
	if password != "" {
		data["password"] = password
	}
	writeMsg(writer, "joinRace", data)
	logMessage(1, "Sent joinRace:", name, key)
}

func sendThrottle(writer *bufio.Writer, throttle float64) {
	writeMsg(writer, "throttle", throttle)
	logMessage(1, "Sent throttle:", throttle)
}

func sendThrottleTick(writer *bufio.Writer, throttle float64, gameTick int) {
	writeMsgTick(writer, "throttle", throttle, gameTick)
	logMessage(1, "Sent throttleTick:", throttle, gameTick)
}

func sendSwitchLane(writer *bufio.Writer, left bool) {
	if left {
		writeMsg(writer, "switchLane", "Left")
		logMessage(1, "Sent switchLane:", "Left")
	} else {
		writeMsg(writer, "switchLane", "Right")
		logMessage(1, "Sent switchLane:", "Right")
	}
}

func sendTurbo(writer *bufio.Writer) {
	writeMsg(writer, "turbo", "turbo")
	logMessage(1, "Sent turbo")
}

func sendSwitchLaneTick(writer *bufio.Writer, left bool, gameTick int) {
	if left {
		writeMsg(writer, "switchLane", "Left")
		logMessage(1, "Sent switchLaneTick:", "Left", gameTick)
	} else {
		writeMsg(writer, "switchLane", "Right")
		logMessage(1, "Sent switchLaneTick:", "Right", gameTick)
	}
}

func sendCreateRace(writer *bufio.Writer, name, key, trackName, password string, carCount int) {
	data := make(map[string]interface{}, 4)
	data["botId"] = map[string]string{"mame": name, "key": key}
	data["trackName"] = trackName
	data["password"] = password
	data["carCount"] = carCount
	writeMsg(writer, "createRace", data)
	logMessage(1, "Sent createRace:", name, key, trackName, password, carCount)
}

func parseJoin(data interface{}) (join Join) {
	vals := data.(map[string]interface{})
	name := vals["name"].(string)
	key := vals["key"].(string)
	return Join{name, key}
}

func parseCarID(data interface{}) (id CarID) {
	vals := data.(map[string]interface{})
	name := vals["name"].(string)
	color := vals["color"].(string)
	return CarID{name, color}
}

func parseYourCar(data interface{}) (car CarID) {
	return parseCarID(data)
}

func parsePiece(data interface{}) (piecedata TPiece) {
	vals := data.(map[string]interface{})
	if d, ok := vals["length"]; ok {
		length := d.(float64)
		s := false
		if sdat, ok := vals["switch"]; ok {
			s = sdat.(bool)
		}
		return TPiece{length, 0.0, 0.0, false, s}
	}
	radius := vals["radius"].(float64)
	angle := vals["angle"].(float64)
	s := false
	if sdat, ok := vals["switch"]; ok {
		s = sdat.(bool)
	}
	return TPiece{0.0, radius, angle, true, s}
}

func parseTrack(data interface{}) (trackdata Track) {
	vals := data.(map[string]interface{})
	id := vals["id"].(string)
	name := vals["name"].(string)
	piecesdata := vals["pieces"].([]interface{})
	pieces := make([]TPiece, len(piecesdata))
	for i, p := range piecesdata {
		pieces[i] = parsePiece(p)
	}
	lanesdata := vals["lanes"].([]interface{})
	lanes := make([]float64, len(lanesdata))
	for _, ln := range lanesdata {
		v := ln.(map[string]interface{})
		lanes[int(v["index"].(float64))] = v["distanceFromCenter"].(float64)
	}
	return Track{id, name, pieces, lanes}
}

func parseCar(data interface{}) (cardata Car) {
	vals := data.(map[string]interface{})
	id := parseCarID(vals["id"])
	dims := vals["dimensions"].(map[string]interface{})
	length := dims["length"].(float64)
	width := dims["width"].(float64)
	guideFlagPosition := dims["guideFlagPosition"].(float64)
	return Car{id, length, width, guideFlagPosition}
}

func parseGameInit(data interface{}) (racedata Race) {
	vals := data.(map[string]interface{})["race"].(map[string]interface{})
	track := parseTrack(vals["track"])
	carsdata := vals["cars"].([]interface{})
	cars := make([]Car, len(carsdata))
	for i, cardata := range carsdata {
		cars[i] = parseCar(cardata)
	}
	session := vals["raceSession"].(map[string]interface{})
	laps := -1
	maxLapTimeMs := 60000
	quickRace := false
	qualifying := false
	duration := -1
	if _, ok := session["laps"]; ok {
		laps = int(session["laps"].(float64))
		maxLapTimeMs = int(session["maxLapTimeMs"].(float64))
		quickRace = session["quickRace"].(bool)
	} else {
		qualifying = true
		duration = int(session["durationMs"].(float64)*60.0/1000.0 + 0.5)
	}
	return Race{track, cars, laps, maxLapTimeMs, quickRace, qualifying, duration}
}

func parseCarPosition(data interface{}) (position CarPosition) {
	vals := data.(map[string]interface{})
	id := parseCarID(vals["id"])
	angle := vals["angle"].(float64)
	piecePosition := vals["piecePosition"].(map[string]interface{})
	pieceIndex := int(piecePosition["pieceIndex"].(float64))
	inPieceDistance := piecePosition["inPieceDistance"].(float64)
	lane := piecePosition["lane"].(map[string]interface{})
	startLaneIndex := int(lane["startLaneIndex"].(float64))
	endLaneIndex := int(lane["endLaneIndex"].(float64))
	lap := int(piecePosition["lap"].(float64))
	return CarPosition{id, angle, pieceIndex, inPieceDistance, startLaneIndex, endLaneIndex, lap}
}

func parseCarPositions(data interface{}) (positions []CarPosition) {
	vals := data.([]interface{})
	positions = make([]CarPosition, len(vals))
	for i, car := range vals {
		positions[i] = parseCarPosition(car)
	}
	return positions
}

func parseTurboAvailable(data interface{}) (turbo Turbo) {
	vals := data.(map[string]interface{})
	duration := int(vals["turboDurationTicks"].(float64))
	return Turbo{vals["turboFactor"].(float64), duration, 3 * duration}
}

func parseLapData(data interface{}) (lapdata LapData) {
	vals := data.(map[string]interface{})
	if laps, ok := vals["laps"]; ok {
		lapdata.Lap = int(laps.(float64))
	} else if lap, ok := vals["lap"]; ok {
		lapdata.Lap = int(lap.(float64))
	}
	if ticks, ok := vals["ticks"]; ok {
		lapdata.Ticks = int(ticks.(float64))
	}
	if millis, ok := vals["millis"]; ok {
		lapdata.Millis = int(millis.(float64))
	}
	return lapdata
}

func parseCarResult(data interface{}) (result CarResult) {
	vals := data.(map[string]interface{})
	car := parseCarID(vals["car"])
	lapdata := parseLapData(vals["result"])
	return CarResult{car, lapdata}
}

func parseGameEnd(data interface{}) (end GameEnd) {
	vals := data.(map[string]interface{})
	temp := vals["results"].([]interface{})
	results := make([]CarResult, len(temp))
	for i, r := range temp {
		results[i] = parseCarResult(r)
	}
	temp = vals["bestLaps"].([]interface{})
	bestLaps := make([]CarResult, len(temp))
	for i, r := range temp {
		bestLaps[i] = parseCarResult(r)
	}
	return GameEnd{results, bestLaps}
}

func parseCrash(data interface{}) (car CarID) {
	return parseCarID(data)
}

func parseSpawn(data interface{}) (car CarID) {
	return parseCarID(data)
}

func parseLapFinished(data interface{}) (lap FinishedLap) {
	vals := data.(map[string]interface{})
	car := parseCarID(vals["car"])
	laptime := parseLapData(vals["lapTime"])
	racetime := parseLapData(vals["raceTime"])
	ranking := vals["ranking"].(map[string]interface{})
	overall := int(ranking["overall"].(float64))
	fastestLap := int(ranking["fastestLap"].(float64))
	return FinishedLap{car, laptime, racetime, overall, fastestLap}
}

func parseDnf(data interface{}) (dnf DNF) {
	vals := data.(map[string]interface{})
	car := parseCarID(vals["car"])
	reason := vals["reason"].(string)
	return DNF{car, reason}
}

func parseFinish(data interface{}) (car CarID) {
	return parseCarID(data)
}

func parseError(data interface{}) (err string) {
	return data.(string)
}
