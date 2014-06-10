package main

import (
	"math"
)

func DoTurbo(TDat TrackData, newPos map[string]BotCarPosition, Color string, Plan []PlanSwitch, debug int) bool {
	Pos := newPos[Color]
	if Pos.TurboAvailable.Duration > 0 && Pos.TurboEnabled.Cooldown == 0 {
		curPos := Pos
		duration := Pos.TurboAvailable.Duration
		factor := Pos.TurboAvailable.Factor
		logMessage(1, "Debug turbo:", Pos.Lap, TDat.Laps, canFinish(TDat, curPos, Plan, factor, duration, duration/2+1, 0))
		if Pos.Lap == TDat.Laps-1 && canFinish(TDat, curPos, Plan, factor, duration, duration/2+1, 0) {
			logMessage(debug, "Simulating turbo...")
			curPos = getNextPosThrottle(TDat, Pos, Color, Plan, 1)
			curPos.TurboEnabled = curPos.TurboAvailable
			crash, t, curPos := PartialThrottleSimulate(TDat, curPos, Color, Plan, duration/2+1, 1)
			if !crash && hasEnded(TDat, curPos, int(t+0.5)+1) {
				logMessage(1, "Predicted turbo finish without crash:", curPos, t)
				curPos2 := getNextPosThrottle(TDat, Pos, Color, Plan, 1)
				curPos2.TurboEnabled = curPos2.TurboAvailable
				curPos2 = getNextPosThrottle(TDat, curPos2, Color, Plan, 2)
				crash2, t2, curPos2 := PartialThrottleSimulate(TDat, curPos2, Color, Plan, duration/2, 2)
				if !crash2 && hasEnded(TDat, curPos2, int(t+0.5)+2) {
					if t2+1.0 < t {
						logMessage(debug, "Waiting turn for ending turbo:", curPos2, t2)
						return false
					}
					logMessage(debug, "Ending turbo time:", curPos2, t2)
					return true
				}
				logMessage(debug, "Ending turbo time", crash2, t2)
				return true
			}
			logMessage(debug, "Predicting turbo crash or no quick end", curPos, crash, t)
		}
		if !TDat.Pieces[(Pos.PieceIndex+1)%len(TDat.Pieces)].IsBend() {
			sLen := 0.0
			cr := 0.0
			piece := (Pos.PieceIndex + 1) % len(TDat.Pieces)
			for i := piece; !TDat.Pieces[i].IsBend() || cr < 0.006; i = (i + 1) % len(TDat.Pieces) {
				cr += math.Pow(TDat.Pieces[i].InvRad(TDat.Lanes[Pos.EndLane], TDat.Lanes[Pos.EndLane], 0.0), 2.0) * TDat.Pieces[i].Distance(TDat.Lanes[Pos.EndLane], TDat.Lanes[Pos.EndLane])
				if cr > 0.006 {
					break
				}
				sLen += TDat.Pieces[i].Distance(TDat.Lanes[Pos.EndLane], TDat.Lanes[Pos.EndLane])
				//logMessage(debug, "Turbo debuggging:", i, sLen, cr)
			}
			for x, i := piece, (piece+1)%len(TDat.Pieces); i != piece; x, i = (x+1)%len(TDat.Pieces), (i+1)%len(TDat.Pieces) {
				if TDat.Pieces[x].IsBend() && !TDat.Pieces[i].IsBend() {
					if !TDat.Qualifying && Pos.Lap == TDat.Laps-1 && i == 0 {
						break
					}
					s := 0.0
					cr = 0.0
					for j := i; !TDat.Pieces[j].IsBend() || cr < 0.006; j = (j + 1) % len(TDat.Pieces) {
						if !TDat.Qualifying && Pos.Lap == TDat.Laps-1 && j == 0 {
							break
						}
						cr += math.Pow(TDat.Pieces[j].InvRad(TDat.Lanes[Pos.EndLane], TDat.Lanes[Pos.EndLane], 0.0), 2.0) * TDat.Pieces[i].Distance(TDat.Lanes[Pos.EndLane], TDat.Lanes[Pos.EndLane])
						if cr > 0.006 {
							break
						}
						s += TDat.Pieces[j].Distance(TDat.Lanes[Pos.EndLane], TDat.Lanes[Pos.EndLane])
						//logMessage(debug, "Turbo debuggging:", j, s, cr)
					}
					//logMessage(debug, "Straight length:", i, s)
					if s > sLen {
						logMessage(debug, "Spotted better straight:", i, s)
						return false
					}
				}
			}
			curPos = getNextPosThrottle(TDat, Pos, Color, Plan, 1)
			curPos.TurboEnabled = curPos.TurboAvailable
			logMessage(debug, "Simulating straight turbo...")
			crash, t, _ := PartialThrottleSimulate(TDat, curPos, Color, Plan, duration/2+1, 1.0)
			logMessage(debug, "Straight length:", sLen, crash, t)
			if !crash {
				logMessage(debug, "Predicted turbo straight without crash:", curPos, t)
				curPos2 := getNextPosThrottle(TDat, Pos, Color, Plan, 1)
				curPos2 = getNextPosThrottle(TDat, curPos2, Color, Plan, 2)
				crash2, t2, curPos2 := PartialThrottleSimulate(TDat, curPos2, Color, Plan, duration/2, 2)
				if !crash2 {
					if t2+1.0 < t {
						logMessage(debug, "Waiting turn for straight turbo:", curPos2, t2)
						return false
					}
					logMessage(debug, "Straight turbo time:", curPos2, t2)
					return true
				}
				logMessage(debug, "Straight turbo time", crash2, t2)
				return true
			}
		}
	}
	return false
}

func PartialThrottleSimulate(TDat TrackData, Pos BotCarPosition, Color string, Plan []PlanSwitch, turns, tshift int) (bool, float64, BotCarPosition) {
	turboLeft := Pos.TurboEnabled.Duration
	t := 0.0
	curPos := Pos
	for left := 0; !hasEnded(TDat, curPos, left+tshift) && left < turns; left++ {
		if math.Abs(curPos.Angle) > TDat.Params.MaxAngle {
			return true, t, curPos
		}
		curPos = getNextPosThrottle(TDat, curPos, Color, Plan, tshift+left)
		if turboLeft > 0 {
			turboLeft--
		}
		t += 1.0
	}
	if curPos.Lap == TDat.Laps {
		return false, t - curPos.InPieceDistance/curPos.Speed, curPos
	}
	return false, t, curPos
}

func getNextPosThrottle(TDat TrackData, Pos BotCarPosition, Color string, Plan []PlanSwitch, timeShift int) BotCarPosition {
	return greedyThrottle(TDat, Pos, Plan, timeShift)
}

func canFinish(TDat TrackData, Pos BotCarPosition, Plan []PlanSwitch, factor float64, left, turns, tshift int) bool {
	j := turns - left
	curPos := Pos
	if j > 0 {
		for i := 0; i < left; i++ {
			curPos = getNextPosSwitch(TDat, factor, Plan, curPos)
			if hasEnded(TDat, curPos, i+tshift+1) {
				return true
			}
		}
		for i := 0; i < j; i++ {
			curPos = getNextPosSwitch(TDat, 1.0, Plan, curPos)
			if hasEnded(TDat, curPos, i+tshift+1+left) {
				return true
			}
		}
		return false
	}
	for i := 0; i < turns; i++ {
		curPos = getNextPosSwitch(TDat, factor, Plan, curPos)
		if hasEnded(TDat, curPos, i+tshift+1) {
			return true
		}
	}
	return false
}

func DoTurbo2(TDat TrackData, newPos map[string]BotCarPosition, Color string, Plan []PlanSwitch, NextTurbo, debug int) bool {
	Pos := newPos[Color]
	if Pos.TurboAvailable.Duration > 0 && Pos.TurboEnabled.Cooldown == 0 {
		bi := 0
		bc := NextTurbo
		bd := Pos.InPieceDistance
		bp := Pos.PieceIndex
		bl := Pos.Lap
		logMessage(debug+1, "Turbo2 Debug Start:", bc, bl, bp, bd, Pos.Angle, fastHasEnded(TDat, &Pos, 0), NextTurbo)
		temp := new(BotCarPosition)
		temp2 := new(BotCarPosition)
		curPos := new(BotCarPosition)
		normPos := new(BotCarPosition)
		*normPos = Pos
		var i, c int
		fastGreedyThrottle(TDat, normPos, temp, temp2, Plan, i)
		for i = 1; i < NextTurbo-Pos.TurboEnabled.Cooldown; i++ {
			*curPos = *normPos
			curPos.TurboEnabled = curPos.TurboAvailable
			fastGreedyThrottle(TDat, normPos, temp, temp2, Plan, i)
			for c = i; c < NextTurbo; c++ {
				fastGreedyThrottle(TDat, curPos, temp, temp2, Plan, c)
				if math.Abs(curPos.Angle) > TDat.Params.MaxAngle {
					break
				}
				if fastHasEnded(TDat, curPos, c) {
					break
				}
			}
			if fastGetMinAngle(TDat, curPos, temp, Plan, c) > TDat.Params.MaxAngle {
				if i == 1 {
					return false
				}
				continue
			}
			if c < bc {
				bi = i
				bc = c
				bl = curPos.Lap
				bp = curPos.PieceIndex
				bd = curPos.InPieceDistance
				logMessage(debug+1, "Turbo2 Debug Found better end lap:", bc, bi, bl, bp, bd, curPos.Angle, fastHasEnded(TDat, curPos, c), NextTurbo, i)
				if i != 1 {
					return false
				}
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
			bi = i
			bl = curPos.Lap
			bp = curPos.PieceIndex
			bd = curPos.InPieceDistance
			logMessage(debug+1, "Turbo2 Debug Found better:", bc, bi, bl, bp, bd, curPos.Angle, fastHasEnded(TDat, curPos, c), NextTurbo)
			if i != 1 {
				return false
			}
		}
		logMessage(debug+1, "Turbo2 Debug Found best:", bc, bi, bl, bp, bd)
		return bi == 1
	}
	return false
}
