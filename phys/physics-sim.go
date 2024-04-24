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
const PanelWattage float64 = 430
const CellSize float64 = 1046 * 1812 * 0.01 //m^2
const SystemVoltage float64 = 70            //TODO: replace with real value

func mphToMps(mph float64) float64 {
	return mph * 0.44704
}

func ftToMeters(ft float64) float64 {
	return ft * 0.3048
}

func jtomAh(joules float64) float64 {
	return joules / (SystemVoltage * 3600 * 0.01)
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
const numTicks = 100

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
func CalcPhysics(routeName string, battery int, targSpeed int, loopName string, loopCount int, startTime string, cpOneClose string, cpTwoClose string, cpThreeClose string, stageClose string) {
	//TODO: currently no way to account for checkpoints. As they are provided day of maybe we could take an input parameter as to the position or distance along route of the checkpoint and manage from there?

	//vehicle, err := dataaccess.GetVehicle("vehicle.json") //TODO: Change vehicle constants to values attained from api
	//if err != nil {
	//	panic(err)
	//}
	const timeLayout = "15:04"

	startT, err := time.Parse(timeLayout, startTime)
	if err != nil {
		panic(err)
	}
	var currTime = startT
	//TODO: implement acceleration curve

	//maxBatteryCapmAh := vehicle.BatteryCapacityMilliamps
	maxBatteryCapmAh := 50000.0 //placeholder
	initialBatteryCapmAh := maxBatteryCapmAh * (float64(battery) * 0.01)

	route, err := dataaccess.LoadRoute(routeName)
	if err != nil {
		panic(err)
	}
	//fmt.Println("Route loaded")

	loop, err := dataaccess.LoadRoute(loopName)
	if err != nil {
		panic(err)
	}
	//fmt.Println("Loop loaded")

	//create step distance to calculate over from the total route length and tick count
	totalRouteLength := 0.0
	for _, section := range route.Sections {
		totalRouteLength = ftToMeters(section.LengthFt)
	}

	//each step is 1/numticks the length of the whole route
	var stepDistance float64 = 1 / float64(numTicks)
	stepDistance *= totalRouteLength

	targSpeedMps := mphToMps(float64(targSpeed))
	//For each section we begin at a complete stop, thus initial velocity and acceleration are 0
	var initialVelo float64 = 1
	currentTickVelo := initialVelo

	facingDirectionRadians := 0.0
	vehicleAccel := 2.0  //placeholder
	vehicleDecel := -3.0 //placeholder

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

	//fmt.Println("Starting simulation")

	sectionsWithLoops := route.Sections

	//Add Loops
	for i := 0; i < loopCount; i++ {
		sectionsWithLoops = append(sectionsWithLoops, loop.Sections...)
	}

	//weather, err := dataaccess.GetWeather(&sectionsWithLoops[0], dataaccess.WeatherDataOptions{RefreshTimeSeconds: 600000000}) //TODO: adjust refresh time
	///if err != nil {
	//	panic(err)
	//}

	for j, section := range sectionsWithLoops {

		loadString := `-\|/`
		fmt.Printf("\r%c Section: %d / %d", loadString[j%4], j+1, len(sectionsWithLoops))

		//fmt.Println("Calculating for section: ", j+1)

		//if j > 50 { //Used to cap number of sections for testing
		//	break
		//}

		//fmt.Println("Fetching weather and traffic data")
		weather, err := dataaccess.GetWeather(&section, dataaccess.WeatherDataOptions{UsingWeatherCache: true, RefreshTimeSeconds: 60000000000000000}) //TODO: adjust refresh time
		if err != nil {
			panic(err)
		}

		//traffic, err := dataaccess.GetTraffic(section, dataaccess.TrafficDataOptions{RefreshRateSeconds: 60}) //TODO: adjust refresh rate
		//if err != nil {
		//	panic(err)
		//}

		facingDirectionRadians = calculateBearing(section.CoordinatesInitial, section.CoordinatesFinal) // direction estimation for section determined by difference between start and end point

		windSpeed = mphToMps(weather.WindSpeedMph)
		windDirectionRadians = weather.WindDirectionDegrees * math.Pi / 180

		sectionSlope := (section.ElevationFinalFt - section.ElevationInitialFt) / section.LengthFt
		sectionMaxSpeed := min(mphToMps(float64(section.SpeedLimitMph)), targSpeedMps)
		//fmt.Println("Section Max Speed: ", sectionMaxSpeed)

		for i := 0.0; i < ftToMeters(section.LengthFt); i += stepDistance {
			//fmt.Println("Calculating for subsection: ", i)
			currMaxSpeed := min(sectionMaxSpeed, mphToMps(60))

			//basic acceleration model
			if math.Abs(currMaxSpeed-currentTickVelo) < 0.1 {
				currentTickAccel = 0
			} else if math.Abs(currMaxSpeed-currentTickVelo) < 0.5 {
				if currentTickVelo < currMaxSpeed {
					currentTickAccel = vehicleAccel
				} else {
					currentTickAccel = vehicleDecel * 0.5
				}
			} else if currentTickVelo < currMaxSpeed {
				currentTickAccel = vehicleAccel
			} else if currentTickVelo > currMaxSpeed {
				currentTickAccel = vehicleDecel
			} else {
				currentTickAccel = 0
			}

			timeToTravel := 0.0
			if currentTickAccel != 0 {
				timeToTravel = (-currentTickVelo + math.Sqrt(math.Pow(currentTickVelo, 2)+2*currentTickAccel*stepDistance)) / currentTickAccel
			} else {
				timeToTravel = stepDistance / currentTickVelo
			}

			//fmt.Println("Time to travel: ", timeToTravel, "s")
			deltaTimeS += timeToTravel
			currTime = currTime.Add(time.Duration(deltaTimeS) * time.Second)

			prevVelo = currentTickVelo

			currentTickVelo += currentTickAccel * timeToTravel
			//fmt.Println("currentTickVelo: ", currentTickVelo, "m/s")
			//fmt.Println("currentTickAccel: ", currentTickAccel, "m/s^2")

			maxAccel = max(maxAccel, currentTickAccel)
			minAccel = min(minAccel, currentTickAccel)
			maxVelo = min(maxVelo, currentTickVelo) //TODO: not sure this is the correct method of setting the max speed, as before it was allowed to go beyond the "max speed" to take decelleration into account (?). Confirm with Jack
			minVelo = min(minVelo, currentTickVelo)

			//TODO: curvature and centripetal force, is this even possible with how we are storing route data?
			var currentTickEnergy = max(0, -CalculateWorkDone(currentTickVelo, stepDistance, sectionSlope, prevVelo, facingDirectionRadians)) //Energy in Joules
			if currentTickEnergy > 0 {
				totalEnergyUsed += currentTickEnergy
			}
			//fmt.Println("Energy used: ", currentTickEnergy, "J")
			//fmt.Println("Total Energy used: ", totalEnergyUsed, "J")

			//energy gain from sun
			instantaneousPower := min(430, SolarConstant*max(0, math.Cos(weather.SolarZenithDegrees*math.Pi/180))*CellEfficiency*(1-(weather.CloudCoverPercentage*0.01))*CellSize) * float64(Cells) * 0.005 //Does not take into account changes in voltage / current from the system or from working in series
			solarEnergyGain := instantaneousPower * timeToTravel

			//fmt.Println("Instantaneous Power: ", instantaneousPower, "W")
			//fmt.Println("Energy gained: ", solarEnergyGain, "J")

			if solarEnergyGain > 0 {
				totalEnergyGained += solarEnergyGain
			}
			//fmt.Println("Total Energy gained: ", totalEnergyGained, "J")

			currBatteryPercent := min(100, ((initialBatteryCapmAh-jtomAh(totalEnergyUsed)+jtomAh(totalEnergyGained))/maxBatteryCapmAh)*100) //TODO: ensure this calculation is correct
			//fmt.Println("Battery percent: ", currBatteryPercent, "%")

			if graphOutput {
				energyUsedPlot = append(energyUsedPlot, plotter.XY{X: deltaTimeS, Y: currentTickEnergy})
				energyGainedPlot = append(energyGainedPlot, plotter.XY{X: deltaTimeS, Y: solarEnergyGain})
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
		outputGraph(energyUsedPlot, "./plots/energyUsed.png")
		outputGraph(energyGainedPlot, "./plots/energyGained.png")
		outputGraph(veloPlot, "./plots/velocity.png")
		outputGraph(accelPlot, "./plots/acceleration.png")
		outputGraph(batteryPlot, "./plots/battery.png")
	}
}
