package types

/*
This type exists to prevent latitude and longitude from being accidentally switched up,
as some external APIs put longitude before latitude.
*/
type Coordinates struct {
	Latitude, Longitude float64
}

/*
Enum representing instructions such as "Turn left".
Based off OpenRouteService API instructions:
https://giscience.github.io/openrouteservice/api-reference/endpoints/directions/instruction-types#instruction-types
*/
type RouteInstruction int

const (
	Left RouteInstruction = iota
	Right
	SharpLeft
	SharpRight
	SlightLeft
	SlightRight
	Straight
	EnterRoundabout
	ExitRoundabout
	UTurn
	Goal
	Depart
	KeepLeft
	KeepRight
)

type RouteSection struct {
	SpeedLimitMph      uint
	LengthFt           float64
	ElevationInitialFt float64
	ElevationFinalFt   float64
	CoordinatesInitial Coordinates
	CoordinatesFinal   Coordinates
	// Instructions are actions such as "turn right on Main Street", stc
	ExitInstruction string
	InstructionCode RouteInstruction // Based on OpenRouteService instruction codes
	Next            *RouteSection    `json:"-"`
	Route           *Route           `json:"-"`
	PositionInRoute int
}

type Route struct {
	Name     string
	Sections []RouteSection
	IsLoop   bool
}

type Weather struct {
	SolarZenithDegrees   float64
	AirTempDegreesF      float64
	CloudCoverPercentage float64
	WindSpeedMph         float64
	// 0 degrees is North, 90 degrees is East
	WindDirectionDegrees        float64
	RainOnGroundInches          float64
	SurfacePressureUnknownUnits float64 // TODO: add units
}

type Traffic struct {
	FlowSpeedMph float64
}

// TODO: Add units
type Vehicle struct {
	SolarPanelPowerWatts     float64
	DragCoefficient          float64
	AccelerationCurve        []float64
	TirePressureInitialPsi   float64 // remove?
	WheelCircumferenceInches float64
	BatteryCapacityMilliamps float64
	CellCount                int
	CellEfficiency           float64
}
