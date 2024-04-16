package main

import (
	"fmt"
	"math"

	"asc-simulation/dataaccess"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

//mph is MILES per HOUR
//mps is METERS per SECOND

const Cells int = 256
const CellEfficiency float64 = 0.227
const CellSize float64 = 1046 * 1812 * 0.01 //m^2

func mphToMps(mph float64) float64 {
	return mph * 0.44704
}

func ftToMeters(ft float64) float64 {
	return ft * 0.3048
}

func calculateBearing(start Coordinate, end Coordinate) float64 {
	lat1 := start.Latitude
	lon1 := start.Longitude
	lat2 := end.Latitude
	lon2 := end.Longitude

	dLon := lon2 - lon1
	y := math.Sin(dLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)
	bearing := math.Atan2(y, x)

	//normalize from [0-2pi]
	bearing = (bearing + 2*math.Pi) % (2 * math.Pi)
	return bearing
}

//Solar constant

const SolarConstant = 1361.0 //W/m^2

// not yet implemented
var windDirectionRadians = 0.0
var windSpeed = 10.0 //placeholder value

// number of points in the graph to compute:
const numTicks = 1000

// input arguments:
// Solver dictates the velo and accel. Sim dicatates energy required to execute and time elasped. Solver constrained by energy, optimized for time.
// initial velocity, initial acceleration, then accel curve params
// 0: initial velocity
// 1: initial acceleration
// 2-4: parabola params
// next 3: parabola params

func CalculateWorkDone(velocity float64, step_distance float64, slope float64, prev_velo float64, facing_direction float64) float64 {
	const carMassKg = 298.0
	const dragCoefficient = 0.1275
	const wheelCircumference = 1.875216

	//change the velocity to make relative to wind for airResistance only
	relativeVelocity := velocity - windSpeed*math.Cos(math.Abs(windDirectionRadians-facing_direction))
	airResistance := dragCoefficient * math.Pow(relativeVelocity, 2)

	//mgsin(theta)
	slope_force := carMassKg * 9.81 * math.Sin(math.Atan(slope)) // slope = tan(Theta)
	net_velo_energy := .5 * carMassKg * (velocity - prev_velo)
	total_force := airResistance + slope_force
	total_work := total_force*step_distance + net_velo_energy

	//Calculating motor efficiency (function of velocity)
	motorRpm := 60 * (velocity / wheelCircumference)
	motorCurrent := (-3*motorRpm - 2700) / 13
	var motorEfficiency float64
	if motorCurrent >= 14 {
		motorEfficiency = 0.9264 + 0.0015*(motorCurrent-14)
	} else {
		motorEfficiency = ((motorCurrent - 1.1) / (motorCurrent - .37)) - .02
	}

	// work motor does is "positive"
	return motorEfficiency * total_work
}

func integrand(x float64, q float64, w float64, e float64, r float64) float64 {
	return 1.0 / (q*math.Pow(x, 3) + w*math.Pow(x, 2) + e*x + r)
}

// simpson calculates the definite integral of a function using Simpson's rule.
// a: the lower limit of integration.
// b: the upper limit of integration.
// q, w, e, r: parameters of the function to be integrated.
// n: the number of subintervals to use in the approximation; should be even.
func simpson(a float64, b float64, q float64, w float64, e float64, r float64, n int) float64 {
	h := (b - a) / float64(n)
	sum := integrand(a, q, w, e, r) + integrand(b, q, w, e, r)

	for i := 1; i < n; i += 2 {
		sum += 4 * integrand(a+float64(i)*h, q, w, e, r)
	}

	for i := 2; i < n-1; i += 2 {
		sum += 2 * integrand(a+float64(i)*h, q, w, e, r)
	}

	return (h / 3) * sum
}

func outputGraph(inputArr plotter.XYs, fileName string) {
	toPlot := plot.New()

	lines, err := plotter.NewLine(inputArr)
	if err != nil {
		panic(err)
	}

	toPlot.Add(lines)

	toPlot.X.Tick.Marker = plot.DefaultTicks{}
	toPlot.Y.Tick.Marker = plot.DefaultTicks{}
	toPlot.Add(plotter.NewGrid())
	if err := toPlot.Save(4*vg.Inch, 4*vg.Inch, fileName); err != nil {
		panic(err)
	}
}

// physics sim should be main program
func calcPhysics(routeName string, battery int, targSpeed int, loopOne int, loopTwo int, cpOneClose string, cpTwoClose string, cpThreeClose string, stageClose string) {
	//TODO: currently no way to account for checkpoints. As they are provided day of maybe we could take an input parameter as to the position or distance along route of the checkpoint and manage from there?

	vehicle, err := dataaccess.GetVehicle("vehicle.json")
	if err != nil {
		panic(err)
	}

	route, err := dataaccess.LoadRoute(routeName)
	if err != nil {
		panic(err)
	}

	//create step distance to calculate over from the total route length and tick count
	totalRouteLength := 0.0
	for _, section := range route.Sections {
		totalRouteLength = ftToMeters(section.LengthFt)
	}
	var stepDistance float64 = 1 / float64(numTicks)
	stepDistance *= totalRouteLength

	targSpeedMps := mphToMps(float64(targSpeed))
	//For each section we begin at a complete stop, thus initial velocity and acceleration are 0
	var initialVelo float64 = 0.0
	currentTickVelo := initialVelo

	facingDirectionRadians := 0.0 //TODO: currently changes in facing direction not implemented, currently estimated by the general direction of the section
	currentTickAccel := 0.0

	prevVelo := initialVelo
	var totalEnergyUsed = 0.0
	var maxAccel, minAccel, maxVelo, minVelo, maxCentripetal float64 = math.Inf(-1), math.Inf(1), math.Inf(-1), targSpeedMps, math.Inf(-1)

	for index, section := range route.Sections {
		weather, err := dataaccess.GetWeather(section, dataaccess.WeatherDataOptions{RefreshRateSeconds: 60}) //TODO implement weather condition calculations
		if err != nil {
			panic(err)
		}

		facingDirectionRadians = calculateBearing(section.CoordinatesInitial, section.CoordinatesFinal)

		windSpeed = mphToMps(weather.WindSpeedMph)

		//TODO: replace dummy values for a and b with actual values
		a := 1.0
		b := 1.0
		c := currentTickAccel

		sectionSlope := (section.ElevationFinalFt - section.ElevationInitialFt) / section.LengthFt

		for i := 0.0; i < ftToMeters(section.LengthFt); i += stepDistance {
			timeToTravel := stepDistance / currentTickVelo
			prevVelo = currentTickVelo
			currentTickAccel = a*math.Pow(i, 2) + b*i + c
			currentTickVelo += currentTickAccel * timeToTravel

			maxAccel = max(maxAccel, currentTickAccel)
			minAccel = min(minAccel, currentTickAccel)
			maxVelo = max(maxVelo, currentTickVelo) //TODO: this should be changed for asc. The max velocity should be the minimum of the target velocity, the speed limit of the section and the flow of traffic.
			minVelo = min(minVelo, currentTickVelo)

			//TODO curvature and centripetal force, is this even possible with how we are storing route data?

			var currentTickEnergy = CalculateWorkDone(currentTickVelo, stepDistance, sectionSlope, prevVelo, facingDirectionRadians)
			totalEnergyUsed += currentTickEnergy

			//energy gain from sun
			solarEnergySqM := SolarConstant * math.Cos(weather.SolarInclinationAngleDegrees) * stepDistance   // TODO include cloud coverage
			solarEnergyGain := min(solarEnergySqM*CellSize*float64(Cells)*CellEfficiency, 430*float64(Cells)) //430 is the maximum energy output per solar cell

			//TODO:Graph Output
		}

		if battery <= 0 {
			fmt.Println("Battery is dead at section", index)
			break
		}
	}
}
