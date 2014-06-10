package main

import (
	"bufio"
	"io"
	"math"
	"net"
)

type Message struct {
	Type string
	Data interface{}
}

type FullBot struct {
	Reader       *bufio.Reader
	Writer       *bufio.Writer
	Color        string
	CarPositions []map[string]BotCarPosition
	TDat         TrackData
	Messages     []Message
	SwitchPlan   []PlanSwitch
	Learned      [3]bool
	CurSwitch    [2]bool
	CurThrottle  float64
	NextTurbo    int
}

func NewFullBot(conn net.Conn) *FullBot {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	return &FullBot{reader, writer, "", []map[string]BotCarPosition{}, TrackData{"", []float64{}, []TPiece{}, 0, false, 0, DefaultParameters}, []Message{}, []PlanSwitch{}, [3]bool{false, false, false}, [2]bool{false, false}, 1.0, 600}
}

func (Bot *FullBot) Run(name, key, trackName string, carCount int, password string) {
	defer func() {
		r := recover()
		if r != nil {
			if r == io.EOF {
				logMessage(1, "Finished run")
			} else {
				panic(r)
			}
		}
	}()
	if trackName == "" && carCount == 1 {
		sendJoin(Bot.Writer, name, key)
	} else {
		sendJoinRace(Bot.Writer, name, key, trackName, carCount, password)
	}
	for {
		msgType, data, tick := readMsg(Bot.Reader)
		Bot.HandleMsg(msgType, data, tick)
	}
}

func (Bot *FullBot) HandleMsg(msgType string, data interface{}, tick int) {
	switch msgType {
	case "join":
		// pass
	case "joinRace":
		// pass
	case "yourCar":
		Bot.Color = parseYourCar(data).Color
	case "gameInit":
		racedata := parseGameInit(data)
		Bot.TDat = TrackData{racedata.RaceTrack.Name, racedata.RaceTrack.Lanes, racedata.RaceTrack.Pieces, racedata.Laps, racedata.Qualifying, racedata.Duration, Bot.TDat.Params}
		logMessage(0, Bot.TDat.Pieces)
		Bot.CarPositions = []map[string]BotCarPosition{}
		Bot.CurSwitch = [2]bool{false, false}
		Bot.CurThrottle = 1.0
	case "gameStart":
		sendThrottle(Bot.Writer, 1.0)
		Bot.CurSwitch = [2]bool{false, false}
		Bot.CurThrottle = 1.0
	case "lapFinished":
		// pass
	case "carPositions":
		Bot.HandlePositions(parseCarPositions(data), tick)
	case "gameEnd":
		P := Bot.TDat.Params
		logMessage(0, "Game end:", data, Bot.TDat.Name, P.X*(1.0-P.D), P.F/math.Sqrt(P.M), P.X*(1.0-P.D)+P.F/math.Sqrt(P.M))
	case "tournamentEnd":
		// pass
	case "turboAvailable":
		Bot.Messages = append(Bot.Messages, Message{msgType, data})
	case "turboStart":
		Bot.Messages = append(Bot.Messages, Message{msgType, data})
	case "turboEnd":
		Bot.Messages = append(Bot.Messages, Message{msgType, data})
	case "crash":
		Bot.Messages = append(Bot.Messages, Message{msgType, data})
	case "spawn":
		Bot.Messages = append(Bot.Messages, Message{msgType, data})
	case "dnf":
		Bot.Messages = append(Bot.Messages, Message{msgType, data})
	case "finish":
		Bot.Messages = append(Bot.Messages, Message{msgType, data})
	case "error":
		logMessage(0, "Error:", data)
	default:
		logMessage(0, "Unexpected msgType:", msgType)
	}
}

func (Bot *FullBot) HandlePositions(positions []CarPosition, tick int) {
	if Bot.TDat.Duration > 0 {
		Bot.TDat.Duration--
	}
	if Bot.NextTurbo > 0 {
		Bot.NextTurbo--
	}
	if len(Bot.CarPositions) == 0 {
		newPos := getBotCarPositions(Bot.TDat, map[string]BotCarPosition{}, positions, Bot.Learned[0])
		newCar := newPos[Bot.Color]
		Bot.CarPositions = append(Bot.CarPositions, newPos)
		if Bot.TDat.Qualifying {
			fakeTDat := Bot.TDat
			fakeTDat.Laps = 50
			best, score := PlanFullSwitch(fakeTDat, newCar.Lap, newCar.PieceIndex, newCar.EndLane)
			logMessage(1, "Switch Plan:", best, score)
			Bot.SwitchPlan = best
		} else {
			fakeTDat := Bot.TDat
			if Bot.TDat.Laps > 200 {
				fakeTDat.Laps = 200
			}
			best, score := PlanFullSwitch(Bot.TDat, newCar.Lap, newCar.PieceIndex, newCar.EndLane)
			logMessage(1, "Switch Plan:", best, score)
			Bot.SwitchPlan = best
		}
		return
	}
	if tick == -1 {
		return
	}
	oldPos := Bot.CarPositions[len(Bot.CarPositions)-1]
	newPos := getBotCarPositions(Bot.TDat, oldPos, positions, Bot.Learned[0])
	remMsg := make([]Message, 0)
	for _, msg := range Bot.Messages {
		switch msg.Type {
		case "crash":
			color := msg.Data.(map[string]interface{})["color"].(string)
			bot := newPos[color]
			bot.Crashed = 400 // Guess 400
			if color == Bot.Color {
				Bot.CurThrottle = 0.0
			}
			newPos[color] = bot
		case "spawn":
			color := msg.Data.(map[string]interface{})["color"].(string)
			bot := newPos[color]
			bot.Crashed = 0
			if color == Bot.Color {
				Bot.CurThrottle = 0.0
			}
			newPos[color] = bot
		case "dnf":
			color := msg.Data.(map[string]interface{})["car"].(map[string]interface{})["color"].(string)
			bot := newPos[color]
			bot.Finished = true
			newPos[color] = bot
		case "finish":
			color := msg.Data.(map[string]interface{})["color"].(string)
			bot := newPos[color]
			bot.Finished = true
			newPos[color] = bot
		default:
			remMsg = append(remMsg, msg)
		}
	}
	Bot.Messages = []Message{}
	for _, msg := range remMsg {
		switch msg.Type {
		case "turboAvailable":
			duration := int(msg.Data.(map[string]interface{})["turboDurationTicks"].(float64))
			factor := msg.Data.(map[string]interface{})["turboFactor"].(float64)
			for color, bot := range newPos {
				if (bot.Crashed == 0) && !bot.Finished && bot.TurboEnabled.Cooldown == 0 {
					bot.TurboAvailable = Turbo{factor, duration + 1, duration + 1}
					newPos[color] = bot
				}
			}
			Bot.NextTurbo = 600
		case "turboStart":
			color := msg.Data.(map[string]interface{})["color"].(string)
			bot := newPos[color]
			bot.TurboEnabled = bot.TurboAvailable
			bot.TurboAvailable = Turbo{1.0, 0, 0}
			newPos[color] = bot
		case "turboEnd":
			color := msg.Data.(map[string]interface{})["color"].(string)
			bot := newPos[color]
			bot.TurboEnabled.Factor = 1.0
			bot.TurboEnabled.Cooldown -= bot.TurboEnabled.Duration
			bot.TurboEnabled.Duration = 0
			newPos[color] = bot
		case "error":
			logMessage(1, "Error:", msg.Data)
		default:
			logMessage(0, "Unexpected msgType:", msg.Type, msg.Data)
		}
	}
	Bot.CarPositions = append(Bot.CarPositions, newPos)
	oldCar := oldPos[Bot.Color]
	newCar := newPos[Bot.Color]
	if (newCar.Crashed > 0) || newCar.Finished {
		sendPing(Bot.Writer)
		return
	}
	if oldCar.PieceIndex != newCar.PieceIndex && Bot.TDat.Pieces[newCar.PieceIndex].IsSwitch() {
		do, dir := DoSwitch(Bot.TDat.Pieces, oldCar.PieceIndex, oldCar.Lap, Bot.SwitchPlan)
		if do != Bot.CurSwitch[0] || (do == Bot.CurSwitch[0] && dir != Bot.CurSwitch[1]) {
			logMessage(1, "Detected change in switch:", do, dir, Bot.CurSwitch[0], Bot.CurSwitch[1])
			if Bot.TDat.Qualifying {
				fakeTDat := Bot.TDat
				fakeTDat.Laps = 50
				best, score := PlanFullSwitch(fakeTDat, newCar.Lap, newCar.PieceIndex, newCar.EndLane)
				logMessage(1, "Switch Plan:", best, score)
				Bot.SwitchPlan = best
			} else {
				fakeTDat := Bot.TDat
				if fakeTDat.Laps > 200 {
					fakeTDat.Laps = 200
				}
				best, score := PlanFullSwitch(fakeTDat, newCar.Lap, newCar.PieceIndex, newCar.EndLane)
				logMessage(1, "Switch Plan:", best, score)
				Bot.SwitchPlan = best
			}
		}
		Bot.CurSwitch = [2]bool{false, false}
	}
	if Bot.TDat.Qualifying && oldCar.Lap < newCar.Lap {
		fakeTDat := Bot.TDat
		fakeTDat.Laps = 50
		best, score := PlanFullSwitch(fakeTDat, newCar.Lap, newCar.PieceIndex, newCar.EndLane)
		logMessage(1, "Switch Plan:", best, score)
		Bot.SwitchPlan = best
	}
	tCar := getNextPosSwitch(Bot.TDat, Bot.CurThrottle, Bot.SwitchPlan, oldCar)
	logMessage(2, "Position:", newCar, "Predicted:", tCar, "Throttle:", Bot.CurThrottle)
	Bot.TDat.Params, Bot.Learned = Learn(Bot.TDat, Bot.Learned, Bot.CarPositions, Bot.Color)
	Bot.TDat.Params.MaxAngle = Bot.TDat.Params.MA
	caution := bumpRisk(Bot.TDat, newPos, Bot.Color, 1)
	if Bot.TDat.Qualifying {
		caution += 15.0
	}
	Bot.TDat.Params.MaxAngle -= caution
	logMessage(2, "Offsetting MaxAngle by:", caution)
	logMessage(2, "Current MaxAngle:", Bot.TDat.Params.MaxAngle)
	nextCar := getNextPosSwitch(Bot.TDat, Bot.CurThrottle, Bot.SwitchPlan, newCar)
	logMessage(2, "Next position at current throttle:", Bot.CurThrottle, nextCar, getMinAngle(Bot.TDat, nextCar, Bot.SwitchPlan, 2))
	if getMinAngle(Bot.TDat, nextCar, Bot.SwitchPlan, 2) < Bot.TDat.Params.MaxAngle || Bot.CurThrottle == 0.0 {
		logMessage(2, "Checking for non-throttle moves:", Bot.CurSwitch)
		do, dir, plan := DoOvertake(Bot.TDat, newPos, Bot.Color, Bot.SwitchPlan, Bot.CurSwitch, 1)
		if do {
			if !(Bot.CurSwitch[0]) || dir != Bot.CurSwitch[1] {
				sendSwitchLane(Bot.Writer, dir)
				Bot.CurSwitch = [2]bool{do, dir}
				logMessage(2, "Setting overtake switch plan:", plan)
				Bot.SwitchPlan = plan
				return
			}
		}
		if !Bot.CurSwitch[0] {
			do, dir := DoSwitch(Bot.TDat.Pieces, newCar.PieceIndex, newCar.Lap, Bot.SwitchPlan)
			if do {
				sendSwitchLane(Bot.Writer, dir)
				Bot.CurSwitch = [2]bool{do, dir}
				return
			}
		}
		turbo := DoTurbo2(Bot.TDat, newPos, Bot.Color, Bot.SwitchPlan, Bot.NextTurbo, 1)
		if turbo {
			sendTurbo(Bot.Writer)
			return
		}
	}
	if !Bot.Learned[0] {
		sendPing(Bot.Writer)
		return
	}
	throttle := DoThrottle2(Bot.TDat, newPos, Bot.Color, Bot.SwitchPlan, 1)
	if int(throttle*100000.0+0.5) != int(Bot.CurThrottle*100000.0+0.5) {
		sendThrottle(Bot.Writer, throttle)
		Bot.CurThrottle = throttle
		return
	}
	sendPing(Bot.Writer)
	return
}
