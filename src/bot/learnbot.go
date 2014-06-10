package main

import (
	"math"
)

func Learn(TDat TrackData, Learned [3]bool, PosHist []map[string]BotCarPosition, Color string) (Parameters, [3]bool) {
	if len(PosHist) < 3 {
		return TDat.Params, Learned
	}
	P := TDat.Params
	n := len(PosHist)
	if !Learned[0] {
		olderCar := PosHist[n-3][Color]
		oldCar := PosHist[n-2][Color]
		newCar := PosHist[n-1][Color]
		x, d := learnVelocityParams(newCar.Speed, [2]float64{olderCar.Speed, oldCar.Speed}, [2]float64{1.0, 1.0})
		P.X = x
		P.D = d
		P.MA = 59.999 - 45.0
		logMessage(0, "Learned velocity parameters:", x, d)
		Learned[0] = true
		P.A = 0.9
	}
	if Learned[0] && !Learned[1] {
		for color, _ := range PosHist[0] {
			aCar := PosHist[n-1][color]
			bCar := PosHist[n-2][color]
			cCar := PosHist[n-3][color]
			aCarValid := aCar.Angle > 0.001 && aCar.DAngle > 0.001 && aCar.Speed > 0.001 && aCar.StartLane == aCar.EndLane && !aCar.Bumped
			bCarValid := bCar.Angle > 0.001 && bCar.DAngle > 0.001 && bCar.Speed > 0.001 && bCar.StartLane == bCar.EndLane && !bCar.Bumped
			cCarValid := cCar.Angle > 0.001 && cCar.DAngle > 0.001 && cCar.Speed > 0.001 && cCar.StartLane == cCar.EndLane && !cCar.Bumped
			if aCarValid && bCarValid && cCarValid {
				if aCar.PieceIndex == cCar.PieceIndex && TDat.Pieces[aCar.PieceIndex].IsBend() {
					curve := TDat.Pieces[aCar.PieceIndex]
					shift := TDat.Lanes[aCar.StartLane]
					if (aCar.DAngle-TDat.Params.A*bCar.DAngle) > 0.0 && (bCar.DAngle-TDat.Params.A*cCar.DAngle) > 0.0 {
						dangles := [3]float64{cCar.DAngle, bCar.DAngle, aCar.DAngle}
						angles := [2]float64{cCar.Angle, bCar.Angle}
						speeds := [2]float64{cCar.Speed, bCar.Speed}
						invrad := curve.InvRad(shift, shift, 0.0)
						s, m, f := learnMostAngleParams(invrad, curve.Direction(), dangles, speeds, angles)
						P.S = s
						P.M = m
						P.F = f
						P.MA = 59.999
						logMessage(0, "Learned most angle parameters:", s, m, f)
						logMessage(0, "Physics:", P, P.X*(1.0-P.D), P.F/math.Sqrt(P.M), P.X*(1.0-P.D)+P.F/math.Sqrt(P.M))
						Learned[1] = true
						Learned[2] = true
						break
					}
				}
			}
		}
	}
	return P, Learned
}

func learnVelocityParams(speed float64, speeds, throts [2]float64) (float64, float64) {
	x := (speeds[0]*speed - speeds[1]*speeds[1]) / (speed*throts[0] - speeds[1]*(throts[0]+throts[1]) + speeds[0]*throts[1])
	d := (speeds[1]*throts[1] - speed*throts[0]) / (speeds[0]*throts[1] - speeds[1]*throts[0])
	return x, d
}

func learnSimpleAngleParams(dangle float64, speeds, angles, dangles [2]float64) (float64, float64) {
	b := dangle
	c := dangles[1]
	d := dangles[0]
	e := speeds[1] * angles[1]
	f := speeds[0] * angles[0]
	a := (c*e - b*f) / (d*e - c*f)
	s := (c*c - b*d) / (d*e - c*f)
	return a, s
}

func learnAdvancedAngleParams(a, s float64, curve TPiece, shift float64, dangles [3]float64, speeds, angles [2]float64) (float64, float64) {
	invrad := curve.InvRad(shift, shift, 0.0)
	s1 := speeds[1]
	s0 := speeds[0]
	g1 := math.Abs((dangles[2] - a*dangles[1] + s*s1*angles[1]))
	g0 := math.Abs((dangles[1] - a*dangles[0] + s*s0*angles[0]))
	f1 := g1 / s1
	f0 := g0 / s0
	m := (f1 - f0) / (s1 - s0) * (f1 - f0) / (s1 - s0) / invrad
	f := (f1*s0 - f0*s1) / (s1 - s0)
	return m, f
}

func learnAdvancedQuickAngleParams(a, s, f float64, curve TPiece, shift float64, dangles [2]float64, speed, angle float64) float64 {
	invrad := curve.InvRad(shift, shift, 0.0)
	g := math.Abs((dangles[1] - a*dangles[0] + s*speed*angle)) / speed
	m := (g + f) * (g + f) / (speed * speed) / invrad
	return m
}

func learnMostAngleParams(invrad float64, dir bool, dangles [3]float64, speeds, angles [2]float64) (float64, float64, float64) {
	//s := 0.00125*x
	//f := 0.3*x
	//a := 0.9
	a := 0.9
	if dir {
		// dangle[i+1] = a*dangle[i] - s*speed[i]*angle[i] + (sqrtm*sqrt(invrad)*speed[i] - f) * speed[i]
		// (dangle[i+1] - a*dangle[i])/speed[i] = sqrtm*sqrt(invrad)*speed[i] - x*(fp + sp*angle[i])
		a0 := math.Sqrt(invrad) * speeds[0]
		b0 := -(0.3 + 0.00125*angles[0])
		c0 := (dangles[1] - a*dangles[0]) / speeds[0]
		a1 := math.Sqrt(invrad) * speeds[1]
		b1 := -(0.3 + 0.00125*angles[1])
		c1 := (dangles[2] - a*dangles[1]) / speeds[1]
		m := math.Pow((c0*b1-b0*c1)/(a0*b1-b0*a1), 2.0)
		x := (c0*a1 - a0*c1) / (b0*a1 - a0*b1)
		return x * 0.00125, m, x * 0.3
	}
	// dangle[i+1] = a*dangle[i] - s*speed[i]*angle[i] - (sqrtm*sqrt(invrad)*speed[i] - f) * speed[i]
	// (dangle[i+1] - a*dangle[i])/speed[i] = -sqrtm*sqrt(invrad)*speed[i] + x*(fp - sp*angle[i])
	a0 := -math.Sqrt(invrad) * speeds[0]
	b0 := 0.3 - 0.00125*angles[0]
	c0 := (dangles[1] - a*dangles[0]) / speeds[0]
	a1 := -math.Sqrt(invrad) * speeds[1]
	b1 := 0.3 - 0.00125*angles[1]
	c1 := (dangles[2] - a*dangles[1]) / speeds[1]
	m := math.Pow((c0*b1-b0*c1)/(a0*b1-b0*a1), 2.0)
	x := (c0*a1 - a0*c1) / (b0*a1 - a0*b1)
	return x * 0.00125, m, x * 0.3
}
