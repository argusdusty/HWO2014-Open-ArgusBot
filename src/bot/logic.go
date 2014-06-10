package main

import (
	"math"
)

const RadFactor = math.Pi / 180.0

type TPiece struct {
	Length float64
	Radius float64
	Angle  float64
	Bend   bool
	Switch bool
}

func (P TPiece) Distance(start, end float64) float64 {
	if P.Bend {
		if start == end {
			if P.Angle < 0.0 {
				return -P.Angle * RadFactor * (P.Radius + start)
			}
			return P.Angle * RadFactor * (P.Radius - start)
		}
		ang := P.Angle
		rad := int(P.Radius)
		sl := int(start)
		el := int(end)
		a := 1
		if ang < 0.0 {
			a = -1
			ang = -ang
		}
		if b, ok := bDist[ang]; ok {
			if s, ok := b[rad-a*sl]; ok {
				if d, ok := s[rad-a*el]; ok {
					return d
				}
			}
		}
		d := BendSwitchLength(ang, rad-a*sl, rad-a*el)
		if _, ok := bDist[ang]; !ok {
			bDist[ang] = map[int]map[int]float64{}
		}
		if _, ok := bDist[ang][rad-a*sl]; !ok {
			bDist[ang][rad-a*sl] = map[int]float64{}
		}
		bDist[ang][rad-a*sl][rad-a*el] = d
		return d
	}
	if start == end {
		return P.Length
	} else if s, ok := sDist[int(P.Length)]; ok {
		if d, ok := s[int(math.Abs(start-end))]; ok {
			return d
		}
	}
	d := StraightSwitchLength(P.Length, math.Abs(start-end))
	if _, ok := sDist[int(P.Length)]; !ok {
		sDist[int(P.Length)] = map[int]float64{}
	}
	sDist[int(P.Length)][int(math.Abs(start-end))] = d
	return d
}

func (P TPiece) IsSwitch() bool { return P.Switch }
func (P TPiece) IsBend() bool   { return P.Bend }

func (P TPiece) InvRad(start, end, dist float64) float64 {
	if P.Bend {
		if start == end {
			if P.Angle < 0.0 {
				return 1.0 / (P.Radius + start)
			}
			return 1.0 / (P.Radius - start)
		}
		piece := int(100.0 * dist / P.Distance(start, end))
		if piece <= 0 {
			return 0.0
		} else if piece > 100 {
			piece = 100
		}
		ang := P.Angle
		rad := int(P.Radius)
		sl := int(start)
		el := int(end)
		a := 1
		if ang < 0.0 {
			a = -1
			ang = -ang
		}
		if b, ok := bIRad[ang]; ok {
			if s, ok := b[rad-a*sl]; ok {
				if d, ok := s[rad-a*el]; ok {
					return d[piece]
				}
			}
		}
		d := ApproxBendRadii(P.Distance(start, end), ang, rad-a*sl, rad-a*el)
		if _, ok := bIRad[ang]; !ok {
			bIRad[ang] = map[int]map[int][100]float64{}
		}
		if _, ok := bIRad[ang][rad-a*sl]; !ok {
			bIRad[ang][rad-a*sl] = map[int][100]float64{}
		}
		bIRad[ang][rad-a*sl][rad-a*el] = d
		return d[piece]
	}
	return 0.0
}

func (P TPiece) Direction() bool { return P.Angle > 0.0 }

var sDist = map[int]map[int]float64{110: map[int]float64{20: 111.87803306256944}, 100: map[int]float64{20: 102.0602749929338}, 99: map[int]float64{20: 101.0804661468}, 90: map[int]float64{20: 92.2816812724387}, 78: map[int]float64{20: 80.61901356381203}, 70: map[int]float64{20: 72.90457277751446}}

var bDist = map[float64]map[int]map[int]float64{}

var bIRad = map[float64]map[int]map[int][100]float64{}

func quad_bez(t, a, b, c float64) float64 {
	return (1.0-t)*(1.0-t)*a + 2.0*(1.0-t)*t*b + t*t*c
}
func cube_bez(t, a, b, c, d float64) float64 {
	return (1.0-t)*quad_bez(t, a, b, c) + t*quad_bez(t, b, c, d)
}

func BendSwitchLength(Angle float64, StartRadius, EndRadius int) float64 {
	ax := 0.0
	ay := 0.0
	bx := float64(StartRadius) - math.Cos(Angle*RadFactor/2.0)*float64(StartRadius+EndRadius)/2.0
	by := -math.Sin(Angle*RadFactor/2.0) * float64(StartRadius+EndRadius) / 2.0
	cx := float64(StartRadius) - math.Cos(Angle*RadFactor)*float64(EndRadius)
	cy := -math.Sin(Angle*RadFactor) * float64(EndRadius)
	bx = 2.0*bx - (ax+cx)/2.0
	by = 2.0*by - (ay+cy)/2.0
	lx := 0.0
	ly := 0.0
	ln := 0.0
	for t := 1; t < 10002; t++ {
		x := quad_bez(float64(t)/10000.0, ax, bx, cx)
		y := quad_bez(float64(t)/10000.0, ay, by, cy)
		xx := x - lx
		yy := y - ly
		ln += math.Sqrt(xx*xx + yy*yy)
		lx = x
		ly = y
	}
	logMessage(1, "Bend switch length:", Angle, StartRadius, EndRadius, ln)
	return ln
}

func getInvRad(ax, ay, bx, by, cx, cy float64) float64 {
	abx := bx - ax
	bcx := cx - bx
	acx := cx - ax
	aby := by - ay
	bcy := cy - by
	acy := cy - ay
	a := math.Sqrt((abx*abx + aby*aby) * (bcx*bcx + bcy*bcy) * (acx*acx + acy*acy))
	b := 2.0 * math.Abs(ax*by+bx*cy+cx*ay-ax*cy-bx*ay-cx*by)
	return b / a
}

func ApproxBendRadii(Length, Angle float64, StartRadius, EndRadius int) [100]float64 {
	ax := 0.0
	ay := 0.0
	bx := float64(StartRadius) - math.Cos(Angle*RadFactor/2.0)*float64(StartRadius+EndRadius)/2.0
	by := -math.Sin(Angle*RadFactor/2.0) * float64(StartRadius+EndRadius) / 2.0
	cx := float64(StartRadius) - math.Cos(Angle*RadFactor)*float64(EndRadius)
	cy := -math.Sin(Angle*RadFactor) * float64(EndRadius)
	bx = 2.0*bx - (ax+cx)/2.0
	by = 2.0*by - (ay+cy)/2.0
	lx := 0.0
	ly := 0.0
	ln := 0.0
	c := 0
	lpt := 0
	llpt := 0
	ir := [100]float64{}
	for t := 1; t < 10002; t++ {
		x := quad_bez(float64(t)/10000.0, ax, bx, cx)
		y := quad_bez(float64(t)/10000.0, ay, by, cy)
		xx := x - lx
		yy := y - ly
		ln += math.Sqrt(xx*xx + yy*yy)
		lx = x
		ly = y
		if ln >= Length/100.0 {
			ln -= Length / 100.0
			x1 := quad_bez(float64(llpt)/10000.0, ax, bx, cx)
			y1 := quad_bez(float64(llpt)/10000.0, ay, by, cy)
			x2 := quad_bez(float64(lpt)/10000.0, ax, bx, cx)
			y2 := quad_bez(float64(lpt)/10000.0, ay, by, cy)
			x3 := quad_bez(float64(t)/10000.0, ax, bx, cx)
			y3 := quad_bez(float64(t)/10000.0, ay, by, cy)
			ir[c] = getInvRad(x1, y1, x2, y2, x3, y3)
			logMessage(3, "Radii approximation:", Length, Angle, StartRadius, EndRadius, c, ir[c], 1.0/ir[c])
			c++
			if c == 100 {
				break
			}
			llpt = lpt
			lpt = t
		}
	}
	if c == 99 {
		x1 := quad_bez(float64(llpt)/10000.0, ax, bx, cx)
		y1 := quad_bez(float64(llpt)/10000.0, ay, by, cy)
		x2 := quad_bez(float64(lpt)/10000.0, ax, bx, cx)
		y2 := quad_bez(float64(lpt)/10000.0, ay, by, cy)
		x3 := quad_bez(1.0, ax, bx, cx)
		y3 := quad_bez(1.0, ay, by, cy)
		ir[99] = getInvRad(x1, y1, x2, y2, x3, y3)
	}
	ir[0] = 0.0
	logMessage(2, "Radii approximation:", Length, Angle, StartRadius, EndRadius, ir)
	return ir
}

func StraightSwitchLength(Length, LaneDelta float64) float64 {
	ax := 0.0
	ay := 0.0
	bx := LaneDelta * 0.1
	by := Length * 0.25
	cx := LaneDelta * 0.875
	cy := Length * 0.75
	dx := LaneDelta
	dy := Length
	lx := 0.0
	ly := 0.0
	ln := 0.0
	for t := 1; t < 10002; t++ {
		x := cube_bez(float64(t)/10000.0, ax, bx, cx, dx)
		y := cube_bez(float64(t)/10000.0, ay, by, cy, dy)
		xx := x - lx
		yy := y - ly
		ln += math.Sqrt(xx*xx + yy*yy)
		lx = x
		ly = y
	}
	logMessage(1, "Straight switch length:", Length, LaneDelta, ln)
	return ln
}

type Turbo struct {
	Factor   float64
	Duration int
	Cooldown int
}

type BotCarPosition struct {
	Angle           float64
	DAngle          float64
	PieceIndex      int
	InPieceDistance float64
	StartLane       int
	EndLane         int
	Lap             int
	Speed           float64
	TurboAvailable  Turbo
	TurboEnabled    Turbo
	Crashed         int // Time left until spawn - guess 400
	Finished        bool
	Bumped          bool
}

type Parameters struct {
	MaxAngle float64 // Max Angle parameter - may be temporarily altered
	MA       float64 // True Max Angle Parameter
	X        float64 // Max speed parameter
	D        float64 // Drag parameter
	A        float64 // Angle damping parameter
	S        float64 // Decrement of angle from speed
	M        float64 // Multiplier on force-threshold
	F        float64 // Friction parameter
}

type TrackData struct {
	Name       string
	Lanes      []float64
	Pieces     []TPiece
	Laps       int
	Qualifying bool
	Duration   int
	Params     Parameters
}

var DefaultParameters = Parameters{14.999, 14.999, 10.0, 0.98, 0.9, 0.00125, 0.28125, 0.3}

func getNextDAngle(P Parameters, angle, dangle, speed, invrad float64, dir bool) float64 {
	if invrad == 0.0 {
		return P.A*dangle - P.S*speed*angle
	}
	g := (math.Sqrt(P.M*invrad)*speed - P.F) * speed
	if g < 0.0 {
		return P.A*dangle - P.S*speed*angle
	}
	if !dir {
		g = -g
	}
	return P.A*dangle - P.S*speed*angle + g
}

func getNextSpeed(P Parameters, throttle, speed float64) float64 {
	return (1.0-P.D)*P.X*throttle + P.D*speed
}

func getPosSpeed(TDat TrackData, oldPos, newPos BotCarPosition) float64 {
	speed := newPos.InPieceDistance - oldPos.InPieceDistance
	if newPos.PieceIndex != oldPos.PieceIndex {
		speed += TDat.Pieces[oldPos.PieceIndex].Distance(TDat.Lanes[oldPos.StartLane], TDat.Lanes[oldPos.EndLane])
	}
	return speed
}

func getThrottle(P Parameters, prevSpeed, curSpeed float64) float64 {
	return (curSpeed - P.D*prevSpeed) / ((1.0 - P.D) * P.X)
}

func getNextPos(TDat TrackData, throttle float64, Pos BotCarPosition) BotCarPosition {
	if Pos.TurboEnabled.Duration > 0 {
		throttle *= Pos.TurboEnabled.Factor
	}
	tPiece := TDat.Pieces[Pos.PieceIndex]
	params := TDat.Params
	invrad := tPiece.InvRad(TDat.Lanes[Pos.StartLane], TDat.Lanes[Pos.EndLane], Pos.InPieceDistance)
	dir := tPiece.Direction()
	dangle := getNextDAngle(params, Pos.Angle, Pos.DAngle, Pos.Speed, invrad, dir)
	angle := dangle + Pos.Angle
	speed := getNextSpeed(params, throttle, Pos.Speed)
	dist := Pos.InPieceDistance + speed
	piece := Pos.PieceIndex
	start := Pos.StartLane
	end := Pos.EndLane
	lap := Pos.Lap
	pdist := tPiece.Distance(TDat.Lanes[start], TDat.Lanes[end])
	if dist > pdist {
		dist -= pdist
		piece = (piece + 1) % len(TDat.Pieces)
		start = end
		if piece == 0 {
			lap++
		}
	}
	turboe := Pos.TurboEnabled
	if turboe.Duration > 0 {
		turboe.Duration--
	}
	if turboe.Cooldown > 0 {
		turboe.Cooldown--
	}
	crashed := Pos.Crashed
	if crashed > 0 {
		crashed--
	}
	return BotCarPosition{angle, dangle, piece, dist, start, end, lap, speed, Pos.TurboAvailable, turboe, crashed, Pos.Finished, false}
}

func getBotCarPositions(TDat TrackData, oldPos map[string]BotCarPosition, positions []CarPosition, LearnedV bool) map[string]BotCarPosition {
	newPos := make(map[string]BotCarPosition, len(positions))
	for _, car := range positions {
		oldCar := oldPos[car.ID.Color]
		newPos[car.ID.Color] = BotCarPosition{car.Angle, oldCar.DAngle, car.PieceIndex, car.InPieceDistance, car.StartLaneIndex, car.EndLaneIndex, car.Lap, oldCar.Speed, oldCar.TurboAvailable, oldCar.TurboEnabled, oldCar.Crashed, oldCar.Finished, oldCar.Bumped}
	}
	for color, oldCar := range oldPos {
		newCar := newPos[color]
		newCar.DAngle = newCar.Angle - oldCar.Angle
		speed := getPosSpeed(TDat, oldCar, newCar)
		if LearnedV && (speed < getNextSpeed(TDat.Params, -0.1, oldCar.Speed) || (speed > getNextSpeed(TDat.Params, 1.5, oldCar.Speed) && oldCar.TurboEnabled.Duration == 0) || (speed > getNextSpeed(TDat.Params, 1.5*oldCar.TurboEnabled.Factor, oldCar.Speed) && oldCar.TurboEnabled.Duration > 0)) {
			if speed == 0.0 || (newCar.Crashed > 0) {
				logMessage(2, "Probably crash:", speed, oldCar.Speed, getNextSpeed(TDat.Params, 1.5, oldCar.Speed), getNextSpeed(TDat.Params, -0.1, oldCar.Speed), getNextSpeed(TDat.Params, 1.5*oldCar.TurboEnabled.Factor, oldCar.Speed), newCar.Crashed)
				speed = 0.0
			} else if !newCar.Bumped {
				if len(positions) == 1 {
					// Bad estimate
				} else {
					logMessage(2, "Probably bump:", speed, oldCar.Speed, getNextSpeed(TDat.Params, 1.5, oldCar.Speed), getNextSpeed(TDat.Params, -0.1, oldCar.Speed), getNextSpeed(TDat.Params, 1.5*oldCar.TurboEnabled.Factor, oldCar.Speed))
					speed = oldCar.Speed
					newCar.Bumped = true
				}
			} else {
				newCar.Bumped = false
			}
		}
		newCar.Speed = speed
		if newCar.TurboEnabled.Duration > 0 {
			newCar.TurboEnabled.Duration--
			if newCar.TurboEnabled.Duration == 0 {
				newCar.TurboEnabled.Factor = 1.0
			}
		}
		if newCar.TurboEnabled.Cooldown > 0 {
			newCar.TurboEnabled.Cooldown--
		}
		if newCar.Crashed > 0 {
			newCar.Crashed--
		}
		newPos[color] = newCar
	}
	return newPos
}

func fastGetNextPos(TDat TrackData, throttle float64, Pos *BotCarPosition) {
	if Pos.TurboEnabled.Duration > 0 {
		throttle *= Pos.TurboEnabled.Factor
	}
	tPiece := TDat.Pieces[Pos.PieceIndex]
	sl := TDat.Lanes[Pos.StartLane]
	el := TDat.Lanes[Pos.EndLane]
	params := TDat.Params
	invrad := tPiece.InvRad(sl, el, Pos.InPieceDistance)
	Pos.DAngle = params.A*Pos.DAngle - params.S*Pos.Speed*Pos.Angle
	if invrad > 0.0 {
		g := math.Sqrt(params.M*invrad)*Pos.Speed - params.F
		if g > 0.0 {
			if !tPiece.Direction() {
				g = -g
			}
			Pos.DAngle += g * Pos.Speed
		}
	}
	//Pos.DAngle = getNextDAngle(params, Pos.Angle, Pos.DAngle, Pos.Speed, invrad, tPiece.Direction())
	Pos.Angle += Pos.DAngle
	Pos.Speed = (1.0-params.D)*params.X*throttle + params.D*Pos.Speed
	//Pos.Speed = getNextSpeed(params, throttle, Pos.Speed)
	Pos.InPieceDistance += Pos.Speed
	pdist := tPiece.Distance(sl, el)
	if Pos.InPieceDistance > pdist {
		Pos.InPieceDistance -= pdist
		Pos.PieceIndex++
		Pos.StartLane = Pos.EndLane
		if Pos.PieceIndex == len(TDat.Pieces) {
			Pos.Lap++
			Pos.PieceIndex = 0
		}
	}
	if Pos.TurboEnabled.Duration > 0 {
		Pos.TurboEnabled.Duration--
	}
	if Pos.TurboEnabled.Cooldown > 0 {
		Pos.TurboEnabled.Cooldown--
	}
	if Pos.Crashed > 0 {
		Pos.Crashed--
	}
}

func hasEnded(TDat TrackData, Pos BotCarPosition, timeShift int) bool {
	if TDat.Qualifying {
		if TDat.Duration <= timeShift {
			return true
		}
	} else if Pos.Lap == TDat.Laps {
		return true
	}
	return false
}

func fastHasEnded(TDat TrackData, Pos *BotCarPosition, timeShift int) bool {
	if TDat.Qualifying {
		if TDat.Duration <= timeShift {
			return true
		}
	} else if Pos.Lap == TDat.Laps {
		return true
	}
	return false
}
