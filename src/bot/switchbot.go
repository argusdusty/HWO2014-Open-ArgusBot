package main

import (
	"math"
)

var switchCost = 1.05

type PlanSwitch struct {
	Lap   int
	Piece int
	Dir   bool
}

type SwitchLapCacheType struct {
	B bool
	X []PlanSwitch
	T float64
}

type SwitchFullCacheType struct {
	S int
	X []PlanSwitch
	T float64
}

var switchLapCache = map[string]map[int]SwitchLapCacheType{}
var switchFullCache = map[string]map[int]SwitchFullCacheType{}

func PlanFullSwitch(TDat TrackData, curLap, curPiece, curLane int) ([]PlanSwitch, float64) {
	if curLap >= TDat.Laps {
		return []PlanSwitch{}, 0.0
	}
	if _, ok := switchFullCache[TDat.Name]; !ok {
		switchFullCache[TDat.Name] = map[int]SwitchFullCacheType{}
	}
	if s, ok := switchFullCache[TDat.Name]; ok {
		if a, ok := s[(1<<10)*(TDat.Laps-curLap)+curLane]; curPiece == 0 && ok {
			x := make([]PlanSwitch, len(a.X))
			copy(x, a.X)
			for i := 0; i < len(x); i++ {
				x[i].Lap = a.X[i].Lap + (curLap - a.S)
			}
			t := a.T
			return x, t
		}
	}
	best := []PlanSwitch{}
	score := math.MaxFloat64
	for i := 0; i < len(TDat.Lanes); i++ {
		b, x, t := PlanLapSwitch(TDat, curPiece, curLane, i)
		for i := 0; i < len(x); i++ {
			x[i].Lap = curLap
		}
		if b {
			fx, ft := PlanFullSwitch(TDat, curLap+1, 0, i)
			ft += t
			if ft < score {
				score = ft
				best = append(x, fx...)
			}
		}
	}
	if curPiece == 0 {
		switchFullCache[TDat.Name][(1<<10)*(TDat.Laps-curLap)+curLane] = SwitchFullCacheType{curLap, best, score}
	}
	return best, score
}

func PlanLapSwitch(TDat TrackData, curPiece, curLane, endLane int) (bool, []PlanSwitch, float64) {
	if s, ok := switchLapCache[TDat.Name]; ok {
		if a, ok := s[(1<<20)*curPiece+(1<<10)*curLane+endLane]; ok {
			x := make([]PlanSwitch, len(a.X))
			copy(x, a.X)
			return a.B, x, a.T
		}
	}
	if _, ok := switchLapCache[TDat.Name]; !ok {
		switchLapCache[TDat.Name] = map[int]SwitchLapCacheType{}
	}
	d := 0.0
	c := 0
	n := curPiece
	for i := curPiece + 1; i < len(TDat.Pieces); i++ {
		if TDat.Pieces[i].IsSwitch() {
			if n == curPiece {
				n = i
			}
			c++
		} else if n == curPiece {
			d += TDat.Pieces[i].Distance(TDat.Lanes[curLane], TDat.Lanes[curLane])
		}
	}
	if c == 0 {
		switchLapCache[TDat.Name][(1<<20)*curPiece+(1<<10)*curLane+endLane] = SwitchLapCacheType{curLane == endLane, []PlanSwitch{}, d}
		return curLane == endLane, []PlanSwitch{}, d
	}
	if curLane > endLane {
		if curLane-endLane == c {
			dp := TDat.Pieces[n].Distance(TDat.Lanes[curLane], TDat.Lanes[curLane-1])
			if TDat.Pieces[n].IsBend() {
				dp *= switchCost
			}
			dp += d
			b, x, t := PlanLapSwitch(TDat, n, curLane-1, endLane)
			plan := make([]PlanSwitch, len(x)+1)
			plan[0] = PlanSwitch{0, n, true}
			copy(plan[1:], x)
			switchLapCache[TDat.Name][(1<<20)*curPiece+(1<<10)*curLane+endLane] = SwitchLapCacheType{b, plan, t + dp}
			return b, plan, t + dp
		} else if curLane-endLane < c {
			switchLapCache[TDat.Name][(1<<20)*curPiece+(1<<10)*curLane+endLane] = SwitchLapCacheType{false, []PlanSwitch{}, d}
			return false, []PlanSwitch{}, d
		}
	} else if curLane < endLane {
		if endLane-curLane == c {
			dp := TDat.Pieces[n].Distance(TDat.Lanes[curLane], TDat.Lanes[curLane+1])
			if TDat.Pieces[n].IsBend() {
				dp *= switchCost
			}
			dp += d
			b, x, t := PlanLapSwitch(TDat, n, curLane+1, endLane)
			plan := make([]PlanSwitch, len(x)+1)
			plan[0] = PlanSwitch{0, n, false}
			copy(plan[1:], x)
			switchLapCache[TDat.Name][(1<<20)*curPiece+(1<<10)*curLane+endLane] = SwitchLapCacheType{b, plan, t + dp}
			return b, plan, t + dp
		} else if endLane-curLane > c {
			switchLapCache[TDat.Name][(1<<20)*curPiece+(1<<10)*curLane+endLane] = SwitchLapCacheType{false, []PlanSwitch{}, d}
			return false, []PlanSwitch{}, d
		}
	}
	dp := TDat.Pieces[n].Distance(TDat.Lanes[curLane], TDat.Lanes[curLane])
	dp += d
	poss, best, score := PlanLapSwitch(TDat, n, curLane, endLane)
	score += dp
	if curLane > 0 {
		dp = TDat.Pieces[n].Distance(TDat.Lanes[curLane], TDat.Lanes[curLane-1])
		if TDat.Pieces[n].IsBend() {
			dp *= switchCost
		}
		dp += d
		b, x, t := PlanLapSwitch(TDat, n, curLane-1, endLane)
		t += dp
		if b && t < score {
			best = append([]PlanSwitch{PlanSwitch{0, n, true}}, x...)
			score = t
			poss = true
		}
	}
	if curLane < len(TDat.Lanes)-1 {
		dp = TDat.Pieces[n].Distance(TDat.Lanes[curLane], TDat.Lanes[curLane+1])
		if TDat.Pieces[n].IsBend() {
			dp *= switchCost
		}
		dp += d
		b, x, t := PlanLapSwitch(TDat, n, curLane+1, endLane)
		t += dp
		if b && t < score {
			best = append([]PlanSwitch{PlanSwitch{0, n, false}}, x...)
			score = t
			poss = true
		}
	}
	switchLapCache[TDat.Name][(1<<20)*curPiece+(1<<10)*curLane+endLane] = SwitchLapCacheType{poss, best, score}
	return poss, best, score
}

func DoSwitch(Pieces []TPiece, piece, lap int, Plan []PlanSwitch) (bool, bool) {
	if len(Plan) == 0 {
		return false, false
	}
	i := piece
	j := lap
	i = (i + 1) % len(Pieces)
	if i == 0 {
		j++
	}
	for !Pieces[i].IsSwitch() {
		i = (i + 1) % len(Pieces)
		if i == 0 {
			j++
		}
	}
	for _, s := range Plan {
		if s.Lap == j && s.Piece == i {
			return true, s.Dir
		}
		if s.Lap > j || (s.Lap == j && s.Piece > i) {
			break
		}
	}
	return false, false
}
