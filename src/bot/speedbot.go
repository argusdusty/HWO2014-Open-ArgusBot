package main

import (
	"math"
)

func getMinAngle(TDat TrackData, Pos BotCarPosition, Plan []PlanSwitch, timeShift int) float64 {
	curPos := Pos
	ang := math.Abs(curPos.Angle)
	t := 0
	for curPos.Speed > 2.0 && !hasEnded(TDat, curPos, t+timeShift) {
		t++
		curPos = getNextPosSwitch(TDat, 0.0, Plan, curPos)
		if math.Abs(curPos.Angle) > ang {
			ang = math.Abs(curPos.Angle)
			if ang > TDat.Params.MaxAngle {
				return ang
			}
		}
	}
	angle := curPos.Angle + TDat.Params.A/(1.0-TDat.Params.A)*curPos.DAngle
	if math.Abs(angle) > ang {
		ang = math.Abs(angle)
	}
	return ang
}

func getNextPosSwitch(TDat TrackData, throttle float64, Plan []PlanSwitch, Pos BotCarPosition) BotCarPosition {
	nextPos := getNextPos(TDat, throttle, Pos)
	if nextPos.PieceIndex != Pos.PieceIndex && TDat.Pieces[nextPos.PieceIndex].IsSwitch() {
		do, dir := DoSwitch(TDat.Pieces, Pos.PieceIndex, Pos.Lap, Plan)
		if do {
			if dir && nextPos.StartLane > 0 {
				nextPos.EndLane = nextPos.StartLane - 1
			} else if !dir && nextPos.StartLane < len(TDat.Lanes)-1 {
				nextPos.EndLane = nextPos.StartLane + 1
			}
		}
	}
	return nextPos
}

func fastGetMinAngle(TDat TrackData, Pos, curPos *BotCarPosition, Plan []PlanSwitch, timeShift int) float64 {
	*curPos = *Pos
	ang := math.Abs(curPos.Angle)
	t := 0
	for curPos.Speed > 2.0 && !fastHasEnded(TDat, curPos, t+timeShift) {
		t++
		fastGetNextPosSwitch(TDat, 0.0, Plan, curPos)
		if math.Abs(curPos.Angle) > ang {
			ang = math.Abs(curPos.Angle)
			if ang > TDat.Params.MaxAngle {
				return ang
			}
		}
	}
	angle := curPos.Angle + TDat.Params.A/(1.0-TDat.Params.A)*curPos.DAngle
	if math.Abs(angle) > ang {
		ang = math.Abs(angle)
	}
	return ang
}

func fastGetNextPosSwitch(TDat TrackData, throttle float64, Plan []PlanSwitch, Pos *BotCarPosition) {
	prevPiece := Pos.PieceIndex
	prevLap := Pos.Lap
	fastGetNextPos(TDat, throttle, Pos)
	if Pos.PieceIndex != prevPiece && TDat.Pieces[Pos.PieceIndex].IsSwitch() {
		do, dir := DoSwitch(TDat.Pieces, prevPiece, prevLap, Plan)
		if do {
			if dir && Pos.StartLane > 0 {
				Pos.EndLane = Pos.StartLane - 1
			} else if !dir && Pos.StartLane < len(TDat.Lanes)-1 {
				Pos.EndLane = Pos.StartLane + 1
			}
		}
	}
}

func fastGreedyThrottle(TDat TrackData, Pos, temp, fastCar *BotCarPosition, Plan []PlanSwitch, timeShift int) {
	*fastCar = *Pos
	fastGetNextPosSwitch(TDat, 1.0, Plan, fastCar)
	if fastGetMinAngle(TDat, fastCar, temp, Plan, timeShift+1) < TDat.Params.MaxAngle {
		*Pos = *fastCar
	} else {
		fastGetNextPosSwitch(TDat, 0.0, Plan, Pos)
	}
}

func greedyThrottle(TDat TrackData, Pos BotCarPosition, Plan []PlanSwitch, timeShift int) BotCarPosition {
	fastCar := getNextPosSwitch(TDat, 1.0, Plan, Pos)
	if getMinAngle(TDat, fastCar, Plan, timeShift+1) < TDat.Params.MaxAngle {
		return fastCar
	}
	return getNextPosSwitch(TDat, 0.0, Plan, Pos)
}

func isAhead(aCar, bCar BotCarPosition) bool {
	if aCar.Lap < bCar.Lap {
		return false
	} else if aCar.Lap == bCar.Lap {
		if aCar.PieceIndex < bCar.PieceIndex {
			return false
		} else if aCar.PieceIndex == bCar.PieceIndex {
			if aCar.InPieceDistance < bCar.InPieceDistance {
				return false
			}
		}
	}
	return true
}

func DoThrottle(TDat TrackData, Pos map[string]BotCarPosition, Color string, Plan []PlanSwitch, debug int) float64 {
	steps := 1 << 5
	n := 1 << 7
	bc := steps + 1
	bd := Pos[Color].InPieceDistance
	bp := Pos[Color].PieceIndex
	bl := Pos[Color].Lap
	bt := 0.0
	for i := 0; i <= n; i++ {
		t := float64(n-i) / float64(n)
		curPos := getNextPosSwitch(TDat, t, Plan, Pos[Color])
		if math.Abs(curPos.Angle) > TDat.Params.MaxAngle {
			continue
		}
		c := 1
		for j := 0; j < steps; j++ {
			curPos = greedyThrottle(TDat, curPos, Plan, c)
			c++
			if math.Abs(curPos.Angle) > TDat.Params.MaxAngle {
				break
			}
			if hasEnded(TDat, curPos, c) {
				break
			}
		}
		if getMinAngle(TDat, curPos, Plan, c) > TDat.Params.MaxAngle {
			continue
		}
		if c < bc {
			bc = c
			bt = float64(n-i) / float64(n)
			bl = curPos.Lap
			bp = curPos.PieceIndex
			bd = curPos.InPieceDistance
			//logMessage(debug+1, "Throttle Debug Found better end lap:", bt, bc, bl, bp, bd)
			continue
		}
		if c > bc {
			continue
		}
		if curPos.Lap < bl {
			continue
		} else if curPos.Lap == bl {
			if curPos.PieceIndex < bp {
				continue
			} else if curPos.PieceIndex == bp {
				if curPos.InPieceDistance < bd {
					continue
				}
			}
		}
		bt = float64(n-i) / float64(n)
		bl = curPos.Lap
		bp = curPos.PieceIndex
		bd = curPos.InPieceDistance
		//logMessage(debug+1, "Throttle Debug Found better:", bt, c, bl, bp, bd)
	}
	logMessage(debug, "Throttle Debug Best:", bt, bc, bl, bp, bd)
	return bt
}

func DoThrottle2(TDat TrackData, Pos map[string]BotCarPosition, Color string, Plan []PlanSwitch, debug int) float64 {
	botPos := Pos[Color]
	steps := 1 << 5
	n := 1 << 5
	n2 := 1 << 3
	bc := steps + 2
	bd := Pos[Color].InPieceDistance
	bp := Pos[Color].PieceIndex
	bl := Pos[Color].Lap
	bt := 0.0
	temp := new(BotCarPosition)
	temp2 := new(BotCarPosition)
	curPos := new(BotCarPosition)
	for i := 0; i <= n; i++ {
		for i2 := 0; i2 <= n2; i2++ {
			t := float64(n-i) / float64(n)
			t2 := float64(n2-i2) / float64(n2)
			*curPos = botPos
			//logMessage(2, "DoThrottle2 Step:", curPos.Speed, curPos.Angle, -1, t, t2, fastGetMinAngle(TDat, curPos, Plan, 0))
			fastGetNextPosSwitch(TDat, t, Plan, curPos)
			//logMessage(2, "DoThrottle2 Step:", curPos.Speed, curPos.Angle, 0, t, t2, fastGetMinAngle(TDat, curPos, Plan, 0))
			if math.Abs(curPos.Angle) > TDat.Params.MaxAngle {
				continue
			}
			fastGetNextPosSwitch(TDat, t2, Plan, curPos)
			//logMessage(2, "DoThrottle2 Step:", curPos.Speed, curPos.Angle, 1, t, t2, fastGetMinAngle(TDat, curPos, Plan, 0))
			if math.Abs(curPos.Angle) > TDat.Params.MaxAngle {
				continue
			}
			c := 2
			for j := 0; j < steps; j++ {
				fastGreedyThrottle(TDat, curPos, temp, temp2, Plan, c)
				//logMessage(2, "DoThrottle2 Step:", curPos.Speed, curPos.Angle, c, t, t2, fastGetMinAngle(TDat, curPos, Plan, 0))
				c++
				if math.Abs(curPos.Angle) > TDat.Params.MaxAngle {
					break
				}
				if fastHasEnded(TDat, curPos, c) {
					break
				}
			}
			if fastGetMinAngle(TDat, curPos, temp, Plan, c) > TDat.Params.MaxAngle {
				continue
			}
			if c < bc {
				bc = c
				bt = t
				bl = curPos.Lap
				bp = curPos.PieceIndex
				bd = curPos.InPieceDistance
				//logMessage(debug+1, "Throttle2 Debug Found better end lap:", t, t2, bc, bl, bp, bd, curPos.Angle, fastGetMinAngle(TDat, curPos, Plan, c))
				continue
			}
			if c > bc {
				continue
			}
			if curPos.Lap < bl {
				continue
			} else if curPos.Lap == bl {
				if curPos.PieceIndex < bp {
					continue
				} else if curPos.PieceIndex == bp {
					if curPos.InPieceDistance < bd {
						continue
					}
				}
			}
			bt = t
			bl = curPos.Lap
			bp = curPos.PieceIndex
			bd = curPos.InPieceDistance
			//logMessage(debug+1, "Throttle2 Debug Found better:", t, t2, c, bl, bp, bd, curPos.Angle)
		}
	}
	logMessage(debug, "Throttle2 Debug Best:", bt, bc, bl, bp, bd)
	return bt
}
