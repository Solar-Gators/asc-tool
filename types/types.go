package types

/*
This type exists to prevent latitude and longitude from being accidentally switched up,
as some external APIs put longitude before latitude.
*/
type Coordinates struct {
	lat, lon float64
}

type RouteSection struct {
	speedLimitMph      uint
	lengthFt           float64
	elevationInitialFt float64
	elevationFinalFt   float64
	coordinatesInitial Coordinates
	coordinatesFinal   Coordinates
	// Instructions are actions such as "turn right on Main Street", stc
	exitInstruction string
	instructionCode int // Based on OpenRouteService instruction codes
	next            *RouteSection
	positionInRoute int
}

type Route struct {
	sections []RouteSection
	isLoop   bool
}

type Weather struct {
	solarInclinationAngleDegrees float64
	airTemperatureDegreesF       float64
	cloudCover                   float64
	windSpeedMph                 float64
	rainOnGroundInches           float64
}

type Traffic struct {
	flowSpeedMph float64
}

// TODO: Add units
type Vehicle struct {
	solarPanelPower     float64
	dragCoefficient     float64
	accelerationCurve   []float64
	tirePressureInitial float64
}
