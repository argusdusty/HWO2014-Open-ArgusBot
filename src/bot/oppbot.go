package main

import (
	"math"
)

func getOppPerf(TDat TrackData, Pos map[string]BotCarPosition, Color string, maxLaps int) map[string]float64 {
	bestLap := 0
	for _, bot := range Pos {
		if bot.Lap > bestLap {
			bestLap = bot.Lap
		}
	}
	fakeTrack := TDat
	fakeTrack.Laps = bestLap + maxLaps
	BestPos := 0.0
	dist := make(map[string]float64, len(Pos))
	perf := make(map[string]float64, len(Pos))
	_, s := PlanFullSwitch(fakeTrack, 0, 0, 0)
	for color, bot := range Pos {
		if (bot.Crashed > 50) || bot.Finished {
			continue
		}
		_, d := PlanFullSwitch(fakeTrack, bot.Lap, bot.PieceIndex, bot.EndLane)
		d = s - d + bot.InPieceDistance
		if d > BestPos {
			BestPos = d
		}
		dist[color] = d
	}
	for color, _ := range Pos {
		if d, ok := dist[color]; ok {
			perf[color] = 1.0 - math.Pow(1.0-d/BestPos, 1.2)
		}
	}
	return perf
}

func DoOvertake(TDat TrackData, Pos map[string]BotCarPosition, Color string, Plan []PlanSwitch, CurSwitch [2]bool, debug int) (bool, bool, []PlanSwitch) {
	myBot := Pos[Color]
	if TDat.Pieces[(myBot.PieceIndex+1)%len(TDat.Pieces)].IsSwitch() {
		if len(Pos) == 1 {
			return false, false, Plan
		}
		bestLap := 0
		for _, bot := range Pos {
			if bot.Lap > bestLap {
				bestLap = bot.Lap
			}
		}
		fakeTrack := TDat
		fakeTrack.Laps = bestLap + 5
		perf := getOppPerf(TDat, Pos, Color, 2)
		for color, _ := range perf {
			perf[color] *= 0.95 // I might be better than the best
			if Pos[color].TurboEnabled.Duration > 5 {
				perf[color] *= Pos[color].TurboEnabled.Factor
			} else if Pos[color].TurboEnabled.Cooldown == 0 && Pos[color].TurboAvailable.Duration > 0 {
				//perf[color] = math.Sqrt(perf[color]* Pos[color].TurboAvailable.Factor) // Geometric mean
			}
		}
		perf[Color] = 1.0
		if myBot.TurboEnabled.Duration > 5 {
			perf[Color] = myBot.TurboEnabled.Factor
		} else if myBot.TurboEnabled.Cooldown == 0 && myBot.TurboAvailable.Duration > 0 {
			//perf[Color] *= math.Sqrt(myBot.TurboAvailable.Factor) // Geometric mean
		}
		logMessage(debug, "Overtake Debug:", perf)
		lane := myBot.EndLane
		dist := TDat.Pieces[(myBot.PieceIndex+1)%len(TDat.Pieces)].Distance(TDat.Lanes[lane], TDat.Lanes[lane])
		tDist := dist
		j := myBot.Lap
		if myBot.PieceIndex+1 == len(TDat.Pieces) {
			j++
		}
		for i := (myBot.PieceIndex + 2) % len(TDat.Pieces); !TDat.Pieces[(i+1)%len(TDat.Pieces)].IsSwitch(); i = (i + 1) % len(TDat.Pieces) {
			if i == 0 {
				j++
			}
			dist += TDat.Pieces[i].Distance(TDat.Lanes[lane], TDat.Lanes[lane])
		}
		mnCost := 0.0
		i := (myBot.PieceIndex + 1) % len(TDat.Pieces)
		for !TDat.Pieces[(i+1)%len(TDat.Pieces)].IsSwitch() {
			for color, bot := range Pos {
				if p, ok := perf[color]; ok {
					if p == 0.0 {
						mnCost = math.MaxFloat64
					} else if bot.PieceIndex == i && bot.EndLane == lane {
						cost := (dist-tDist)/p - dist/perf[Color]
						if cost > mnCost {
							mnCost = cost
						}
					}
				}
			}
			i = (i + 1) % len(TDat.Pieces)
			tDist += TDat.Pieces[i].Distance(TDat.Lanes[lane], TDat.Lanes[lane])
		}
		bCost := mnCost
		nDist := dist
		nLane := lane
		bLane := lane
		j = myBot.Lap
		if myBot.PieceIndex+1 == len(TDat.Pieces) {
			j++
		}
		bPlan, nd := PlanFullSwitch(fakeTrack, j, (myBot.PieceIndex+1)%len(TDat.Pieces), lane)
		if CurSwitch[0] == true {
			if nLane != 0 && nLane != len(TDat.Lanes)-1 {
				bCost = math.MaxFloat64
			}
		}
		if getMinAngle(TDat, myBot, bPlan, 0) > TDat.Params.MaxAngle { // going to crash
			bCost = math.MaxFloat64
		}
		logMessage(debug, "Debug overtake:", "Calculated no switch cost:", mnCost, dist, nd, lane, bPlan, bCost)
		if nLane > 0 {
			lane := myBot.EndLane - 1
			dist := TDat.Pieces[(myBot.PieceIndex+1)%len(TDat.Pieces)].Distance(TDat.Lanes[nLane], TDat.Lanes[lane])
			tDist := dist
			j := myBot.Lap
			if myBot.PieceIndex+1 == len(TDat.Pieces) {
				j++
			}
			for i := (myBot.PieceIndex + 2) % len(TDat.Pieces); !TDat.Pieces[(i+1)%len(TDat.Pieces)].IsSwitch(); i = (i + 1) % len(TDat.Pieces) {
				if i == 0 {
					j++
				}
				dist += TDat.Pieces[i].Distance(TDat.Lanes[lane], TDat.Lanes[lane])
			}
			mnCost := 0.0
			i := (myBot.PieceIndex + 1) % len(TDat.Pieces)
			for !TDat.Pieces[(i+1)%len(TDat.Pieces)].IsSwitch() {
				for color, bot := range Pos {
					if p, ok := perf[color]; ok {
						if p == 0.0 {
							mnCost = math.MaxFloat64
						} else if bot.PieceIndex == i && bot.EndLane == lane {
							cost := (dist-tDist)/p - dist/perf[Color]
							if cost > mnCost {
								mnCost = cost
							}
						}
					}
				}
				i = (i + 1) % len(TDat.Pieces)
				tDist += TDat.Pieces[i].Distance(TDat.Lanes[lane], TDat.Lanes[lane])
			}
			j = myBot.Lap
			if myBot.PieceIndex+1 == len(TDat.Pieces) {
				j++
			}
			plan, d := PlanFullSwitch(fakeTrack, j, (myBot.PieceIndex+1)%len(TDat.Pieces), lane)
			plan = append([]PlanSwitch{PlanSwitch{j, (myBot.PieceIndex + 1) % len(TDat.Pieces), true}}, plan...)
			mnCost += d - nd + dist - nDist
			if getMinAngle(TDat, myBot, plan, 0) > TDat.Params.MaxAngle && !(CurSwitch[0] && CurSwitch[1]) { // going to crash
				mnCost = math.MaxFloat64
			}
			if mnCost < bCost {
				bCost = mnCost
				bPlan = plan
				bLane = lane
			}
			logMessage(debug, "Debug overtake:", "Calculated left switch cost:", mnCost, dist, d, lane, plan, bPlan)
		}
		if nLane < len(TDat.Lanes)-1 {
			lane := myBot.EndLane + 1
			dist := TDat.Pieces[(myBot.PieceIndex+1)%len(TDat.Pieces)].Distance(TDat.Lanes[nLane], TDat.Lanes[lane])
			tDist := dist
			j := myBot.Lap
			if myBot.PieceIndex+1 == len(TDat.Pieces) {
				j++
			}
			for i := (myBot.PieceIndex + 2) % len(TDat.Pieces); !TDat.Pieces[(i+1)%len(TDat.Pieces)].IsSwitch(); i = (i + 1) % len(TDat.Pieces) {
				if i == 0 {
					j++
				}
				dist += TDat.Pieces[i].Distance(TDat.Lanes[lane], TDat.Lanes[lane])
			}
			mnCost := 0.0
			i := (myBot.PieceIndex + 1) % len(TDat.Pieces)
			for !TDat.Pieces[(i+1)%len(TDat.Pieces)].IsSwitch() {
				for color, bot := range Pos {
					if p, ok := perf[color]; ok {
						if p == 0.0 {
							mnCost = math.MaxFloat64
						} else if bot.PieceIndex == i && bot.EndLane == lane {
							cost := (dist-tDist)/p - dist/perf[Color]
							if cost > mnCost {
								mnCost = cost
							}
						}
					}
				}
				i = (i + 1) % len(TDat.Pieces)
				tDist += TDat.Pieces[i].Distance(TDat.Lanes[lane], TDat.Lanes[lane])
			}
			j = myBot.Lap
			if myBot.PieceIndex+1 == len(TDat.Pieces) {
				j++
			}
			plan, d := PlanFullSwitch(fakeTrack, j, (myBot.PieceIndex+1)%len(TDat.Pieces), lane)
			plan = append([]PlanSwitch{PlanSwitch{j, (myBot.PieceIndex + 1) % len(TDat.Pieces), false}}, plan...)
			mnCost += d - nd + dist - nDist
			if getMinAngle(TDat, myBot, plan, 0) > TDat.Params.MaxAngle && !(CurSwitch[0] && !CurSwitch[1]) { // going to crash
				mnCost = math.MaxFloat64
			}
			if mnCost < bCost {
				bCost = mnCost
				bPlan = plan
				bLane = lane
			}
			logMessage(debug, "Debug overtake:", "Calculated right switch cost:", mnCost, dist, d, lane, bPlan)
		}
		if bLane < nLane {
			return true, true, bPlan
		} else if bLane > nLane {
			return true, false, bPlan
		} else if nLane == 0 && CurSwitch[0] == true {
			return true, true, bPlan
		} else if nLane == len(TDat.Lanes)-1 && CurSwitch[0] == true {
			return true, false, bPlan
		}
		return false, true, bPlan
	}
	return false, false, Plan
}

func bumpRisk(TDat TrackData, Pos map[string]BotCarPosition, Color string, debug int) float64 {
	me := Pos[Color]
	fakeTrack := TDat
	fakeTrack.Laps = 2
	_, myd := PlanFullSwitch(fakeTrack, 0, me.PieceIndex, me.EndLane)
	myd += TDat.Pieces[me.PieceIndex].Distance(TDat.Lanes[me.StartLane], TDat.Lanes[me.EndLane])
	bd := math.MaxFloat64
	bc := ""
	for color, bot := range Pos {
		if color == Color {
			continue
		}
		if (bot.Crashed > 0) || bot.Finished {
			continue
		}
		if bot.EndLane == me.EndLane {
			_, d := PlanFullSwitch(fakeTrack, 0, bot.PieceIndex, me.EndLane)
			d += TDat.Pieces[bot.PieceIndex].Distance(TDat.Lanes[bot.StartLane], TDat.Lanes[bot.EndLane]) - bot.InPieceDistance
			logMessage(debug, "Bump Risk Opp Debug:", color, d, myd, bd)
			if d > myd && d-myd < bd {
				bd = d - myd
				bc = color
			}
		}
	}
	if bc == "" {
		return 0.0
	}
	perf := getOppPerf(TDat, Pos, Color, 2)
	opp := Pos[bc]
	oppp := perf[bc]
	sd := 1.0 - (1.0-oppp)*(1.0-oppp)
	logMessage(debug, "Bump Risk Debug:", bd, bc, myd, perf, oppp, sd)
	for i := 0; i < 30; i++ {
		me = getNextPos(TDat, me.Speed/TDat.Params.X, me)
		opp = getNextPos(TDat, sd, opp)
		if (me.PieceIndex == opp.PieceIndex && opp.InPieceDistance+40.0 > me.InPieceDistance) || (me.PieceIndex == (opp.PieceIndex+1)%len(TDat.Pieces) && opp.InPieceDistance+40.0-TDat.Pieces[opp.PieceIndex].Distance(TDat.Lanes[opp.StartLane], TDat.Lanes[opp.EndLane]) > me.InPieceDistance) || (me.PieceIndex == (opp.PieceIndex+2)%len(TDat.Pieces) && opp.InPieceDistance+40.0-TDat.Pieces[(opp.PieceIndex+1)%len(TDat.Pieces)].Distance(TDat.Lanes[opp.EndLane], TDat.Lanes[opp.EndLane])-TDat.Pieces[opp.PieceIndex].Distance(TDat.Lanes[opp.StartLane], TDat.Lanes[opp.EndLane]) > me.InPieceDistance) {
			invrad := TDat.Pieces[me.PieceIndex].InvRad(TDat.Lanes[me.StartLane], TDat.Lanes[me.EndLane], me.InPieceDistance)
			temprad := TDat.Pieces[me.PieceIndex].InvRad(TDat.Lanes[me.EndLane], TDat.Lanes[me.EndLane], 0.0)
			if temprad > invrad {
				invrad = temprad
			}
			fBump := opp.Speed * opp.Speed * invrad
			fNorm := me.Speed * me.Speed * invrad
			gBump := math.Max(math.Sqrt(TDat.Params.M*invrad)*opp.Speed*opp.Speed-TDat.Params.F*opp.Speed, 0.0)
			gNorm := math.Max(math.Sqrt(TDat.Params.M*invrad)*me.Speed*me.Speed-TDat.Params.F*me.Speed, 0.0)
			logMessage(debug, "Bump Risk Debug:", me, opp, opp.Speed, me.Speed, i, fBump, fNorm, TDat.Params.F, gBump, gNorm, (gBump-gNorm)*math.Pow(27.0/30.0, float64(i))*6.0)
			return math.Max((gBump-gNorm)*math.Pow(27.0/30.0, float64(i))*6.0, 0.0)
		}
	}
	return 0.0
}
