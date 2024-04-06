package ors

import (
	"asc-simulation/types"
	"encoding/json"
)

/*
Type definitions for the OpenRouteService API.
Reference: https://openrouteservice.org/dev/#/api-docs/v2/directions/{profile}/post
*/

type Engine struct {
	Version   string
	BuildDate string `json:"build_date"`
	GraphDate string `json:"graph_date"`
}

type Metadata struct {
	Id             string
	Attribution    string
	OsmFileMd5Hash string `json:"osm_file_md5_hash"`
	Service        string
	Timestamp      int
	// Query this request is responding to.
	// Left as any & can be converted to a concrete type separately.
	Query         any
	Engine        Engine
	SystemMessage string `json:"system_message"`
}

type Summary struct {
	Distance float64
	Duration float64
	Ascent   float64
	Descent  float64
	Fare     int
}

type Maneuver struct {
	Location      *types.Coordinates // nullable
	BearingBefore int                `json:"bearing_before"`
	BearingAfter  int                `json:"bearing_after"`
}

func (maneuver *Maneuver) MarshalJSON() ([]byte, error) {
	type OrsManeuver struct {
		Location      []float64
		BearingBefore int `json:"bearing_before"`
		BearingAfter  int `json:"bearing_after"`
	}

	location := []float64{}
	if maneuver.Location != nil {
		location = encodeCoordinates(*maneuver.Location)
	}

	orsManeuver := OrsManeuver{
		Location:      location,
		BearingBefore: maneuver.BearingBefore,
		BearingAfter:  maneuver.BearingAfter,
	}

	return json.Marshal(orsManeuver)
}

func (maneuver *Maneuver) UnmarshalJSON(data []byte) error {
	type OrsManeuver struct {
		Location      []float64
		BearingBefore int `json:"bearing_before"`
		BearingAfter  int `json:"bearing_after"`
	}

	var orsManeuver OrsManeuver
	err := json.Unmarshal(data, &orsManeuver)
	if err != nil {
		return err
	}

	maneuver.Location = decodeCoordinates(orsManeuver.Location)
	maneuver.BearingBefore = orsManeuver.BearingBefore
	maneuver.BearingAfter = orsManeuver.BearingAfter

	return nil
}

type Step struct {
	Distance        float64
	Duration        float64
	InstructionType int `json:"type"`
	Instruction     string
	Name            string
	ExitNumber      int   `json:"exit_number"`
	ExitBearings    []int `json:"exit_bearings"`
	WayPoints       []int `json:"way_points"`
	Maneuver        Maneuver
}

type Segment struct {
	Distance     float64
	Duration     float64
	Steps        []Step
	DetourFactor float64
	Percentage   float64
	AvgSpeed     float64
	Ascent       float64
	Descent      float64
}

type Warning struct {
	Code    int
	Message int
}

type Route struct {
	Summary   Summary
	Segments  []Segment
	Bbox      []float64
	Geometry  string
	WayPoints []int `json:"way_points"`
	Warnings  []Warning
	// Extra objects that can be requested from API.
	// Left as any & can be converted to a concrete type separately.
	Extras    any
	Departure string
	Arrival   string
}

type DirectionsResponse struct {
	Metadata Metadata
	Routes   []Route
	Bbox     []float64
}

type DirectionsRequest struct {
	Coordinates [][]float64 `json:"coordinates"`
	Attributes  []string    `json:"attributes"`
	Elevation   bool        `json:"elevation"`
	Maneuvers   bool        `json:"maneuvers"`
	Units       string      `json:"units"`
}

// Exists because ORS puts longitude before latitude
func encodeCoordinates(coordinates types.Coordinates) []float64 {
	return []float64{coordinates.Longitude, coordinates.Latitude}
}

// Returns nil if slice is empty.
// Exists because ORS puts longitude before latitude.
func decodeCoordinates(coordinates []float64) *types.Coordinates {
	if coordinates == nil || len(coordinates) == 0 {
		return nil
	}

	return &types.Coordinates{
		Latitude:  coordinates[1],
		Longitude: coordinates[0],
	}
}
