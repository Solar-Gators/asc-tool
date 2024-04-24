package phys

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"asc-simulation/dataaccess"
	"asc-simulation/types"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

//mph is MILES per HOUR
//mps is METERS per SECOND

const Cells int = 256
const CellEfficiency float64 = 0.227
const CellSize float64 = 1046 * 1812 * 0.01 //m^2
const BatteryVoltage float64 = 48.0         //TODO: replace with real value

func mphToMps(mph float64) float64 {
	return mph * 0.44704
}

func ftToMeters(ft float64) float64 {
	return ft * 0.3048
}

func jtomAh(joules float64) float64 {
	return joules / (BatteryVoltage * 3600 * 0.01)
}

func calculateBearing(start types.Coordinates, end types.Coordinates) float64 {
	lat1 := start.Latitude
	lon1 := start.Longitude
	lat2 := end.Latitude
	lon2 := end.Longitude

	dLon := lon2 - lon1
	y := math.Sin(dLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)
	bearing := math.Atan2(y, x)

	//normalize from [0-2pi]
	bearing = math.Mod(bearing+2*math.Pi, 2*math.Pi)
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

	fmt.Print("Starting Physics Simulation\n")
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
func CalcPhysics(routeName string, battery int, targSpeed int, loopOne int, loopTwo int, startTime string, cpOneClose string, cpTwoClose string, cpThreeClose string, stageClose string) {
	//TODO: currently no way to account for checkpoints. As they are provided day of maybe we could take an input parameter as to the position or distance along route of the checkpoint and manage from there?

	vehicle, err := dataaccess.GetVehicle("vehicle.json") //TODO: Change vehicle constants to values attained from api
	if err != nil {
		panic(err)
	}
	const timeLayout = "15:04"

	startT, err := time.Parse(timeLayout, startTime)
	if err != nil {
		panic(err)
	}
	var currTime = startT
	//TODO: implement acceleration curve

	maxBatteryCapmAh := vehicle.BatteryCapacityMilliamps
	currBatteryCapmAh := maxBatteryCapmAh * (float64(battery) * 0.01)

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

	facingDirectionRadians := 0.0
	vehicleAccel := 5.0  //placeholder
	vehicleDecel := -5.0 //placeholder

	currentTickAccel := 0.0

	prevVelo := initialVelo
	var deltaTimeS = 0.0 //Graph or output time to traverse
	var totalEnergyUsed = 0.0
	var totalEnergyGained = 0.0
	var maxAccel, minAccel, maxVelo, minVelo float64 = math.Inf(-1), math.Inf(1), math.Inf(-1), targSpeedMps

	//Graphing
	graphOutput := true
	var energyUsedPlot plotter.XYs
	var energyGainedPlot plotter.XYs
	var veloPlot plotter.XYs
	var accelPlot plotter.XYs
	var batteryPlot plotter.XYs
	var colorOffsetVar = 0.0
	var trackDrawingVelocities = ""

	//TODO: Handle loops and loop count

	for _, section := range route.Sections {
		weather, err := dataaccess.GetWeather(&section, dataaccess.WeatherDataOptions{RefreshTimeSeconds: 60}) //TODO: adjust refresh time
		if err != nil {
			panic(err)
		}

		traffic, err := dataaccess.GetTraffic(section, dataaccess.TrafficDataOptions{RefreshRateSeconds: 60}) //TODO: adjust refresh rate
		if err != nil {
			panic(err)
		}

		facingDirectionRadians = calculateBearing(section.CoordinatesInitial, section.CoordinatesFinal) // direction estimation for section determined by difference between start and end point

		windSpeed = mphToMps(weather.WindSpeedMph)
		windDirectionRadians = weather.WindDirectionDegrees * math.Pi / 180

		sectionSlope := (section.ElevationFinalFt - section.ElevationInitialFt) / section.LengthFt
		sectionMaxSpeed := min(mphToMps(float64(section.SpeedLimitMph)), targSpeedMps)

		for i := 0.0; i < ftToMeters(section.LengthFt); i += stepDistance {
			currMaxSpeed := min(sectionMaxSpeed, mphToMps(traffic.FlowSpeedMph))

			timeToTravel := stepDistance / currentTickVelo
			deltaTimeS += timeToTravel

			currTime = currTime.Add(time.Duration(deltaTimeS) * time.Second)

			prevVelo = currentTickVelo
			if currentTickVelo < currMaxSpeed {
				currentTickAccel = vehicleAccel
			} else if currentTickVelo > currMaxSpeed {
				currentTickAccel = vehicleDecel
			} else {
				currentTickAccel = 0
			}
			currentTickVelo += currentTickAccel * timeToTravel

			maxAccel = max(maxAccel, currentTickAccel)
			minAccel = min(minAccel, currentTickAccel)
			maxVelo = min(maxVelo, currentTickVelo) //TODO: not sure this is the correct method of setting the max speed, as before it was allowed to go beyond the "max speed" to take decelleration into account (?). Confirm with Jack
			minVelo = min(minVelo, currentTickVelo)

			//TODO: curvature and centripetal force, is this even possible with how we are storing route data?

			var currentTickEnergy = CalculateWorkDone(currentTickVelo, stepDistance, sectionSlope, prevVelo, facingDirectionRadians) //Energy in Joules
			totalEnergyUsed += currentTickEnergy

			//energy gain from sun
			solarEnergySqM := SolarConstant * math.Cos(weather.SolarZenithDegrees) * stepDistance * (1 - (weather.CloudCoverPercentage * 0.01))
			solarEnergyGain := min(solarEnergySqM*CellSize*float64(Cells)*CellEfficiency, 430*float64(Cells)) //430 is the maximum energy output per solar cell TODO: Fix

			totalEnergyGained += solarEnergyGain

			currBatteryPercent := (currBatteryCapmAh - jtomAh(totalEnergyUsed) + jtomAh(totalEnergyGained)) / maxBatteryCapmAh //TODO: ensure this calculation is correct

			if graphOutput {
				energyUsedPlot = append(energyUsedPlot, plotter.XY{X: deltaTimeS, Y: totalEnergyUsed})
				energyGainedPlot = append(energyGainedPlot, plotter.XY{X: deltaTimeS, Y: totalEnergyGained})
				veloPlot = append(veloPlot, plotter.XY{X: deltaTimeS, Y: currentTickVelo})
				accelPlot = append(accelPlot, plotter.XY{X: deltaTimeS, Y: currentTickAccel})
				batteryPlot = append(batteryPlot, plotter.XY{X: deltaTimeS, Y: currBatteryPercent})

			}
			// if statment only needed to prevent printing final point
			if graphOutput && colorOffsetVar/totalRouteLength <= 1.0 {
				//max of 16 units of speed... can change scale later by putting in for denominator
				const redDivisor = 16

				const blue = 0
				const green = 0
				//converts and makes velocity string
				colorOffsetStr := strconv.FormatFloat(colorOffsetVar/totalRouteLength, 'f', 4, 64)

				trackDrawingVelocities += "<stop offset=\"" + colorOffsetStr + "\" style=\"stop-color:rgb(" + strconv.Itoa(int(math.Round(255*currentTickVelo/redDivisor))) + "," + strconv.Itoa(green) + "," + strconv.Itoa(blue) + ");stop-opacity:1\"/>\n"

				colorOffsetVar += stepDistance
			}
		}
	}
	if graphOutput {
		os.MkdirAll("./plots", 0755)
		outputGraph(energyUsedPlot, "energyUsed.png")
		outputGraph(energyGainedPlot, "energyGained.png")
		outputGraph(veloPlot, "velocity.png")
		outputGraph(accelPlot, "acceleration.png")
		outputGraph(batteryPlot, "battery.png")
	}
}
