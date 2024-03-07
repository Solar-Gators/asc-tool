package types

/*
This type exists to prevent latitude and longitude from being accidentally switched up,
as some external APIs put longitude before latitude.
*/
type Coordinates struct {
	Latitude, Longitude float64
}

type RouteSection struct {
	SpeedLimitMph      uint
	LengthFt           float64
	ElevationInitialFt float64
	ElevationFinalFt   float64
	CoordinatesInitial Coordinates
	CoordinatesFinal   Coordinates
	// Instructions are actions such as "turn right on Main Street", stc
	ExitInstruction string
	InstructionCode int // Based on OpenRouteService instruction codes
	Next            *RouteSection
	PositionInRoute int
}

type Route struct {
	Sections []RouteSection
	IsLoop   bool
}

type Weather struct {
	SolarInclinationAngleDegrees float64
	AirTemperatureDegreesF       float64
	CloudCover                   float64
	WindSpeedMph                 float64
	RainOnGroundInches           float64
}

type Traffic struct {
	FlowSpeedMph float64
}

// TODO: Add units
type Vehicle struct {
	SolarPanelPower     float64
	DragCoefficient     float64
	AccelerationCurve   []float64
	TirePressureInitial float64
}
