package dataaccess

import (
	"asc-simulation/dataaccess/ors"
	"asc-simulation/types"
	"encoding/json"
	"errors"
	"math"
	"os"
	"time"

	"github.com/tkrajina/gpxgo/gpx"
)

/*
Creates route files (which is just a .json file) from a .gpx file.
Each route in the .gpx file will be written to a separate .json file in
the output folder.

This function should not be called during a simulation!
It calls external APIs to get additional route data.

This function returns an slice of errors; if it fails at creating a .json
file for one route, it will continue trying to create .json files for all of the
other routes. Always check the returned slice to see if any routes were not written to .json.
*/
func CreateRoutes(inputGpxFilePath string, outputFolder string) []error {
	gpxFile, err := gpx.ParseFile(inputGpxFilePath)
	if err != nil {
		return []error{errors.Join(errors.New("error parsing .gpx file"), err)}
	}

	var routes []types.Route
	var errs []error

	for _, route := range gpxFile.Routes {
		createdRoute, err := createRouteFromGpx(&route)
		if err != nil {
			newErr := errors.Join(
				errors.New("could not get route data for route \""+route.Name+"\""),
				err,
			)
			errs = append(errs, newErr)
			continue
		}

		routes = append(routes, *createdRoute)
	}

	for i := range routes {
		err = writeRouteToFile(&routes[i], outputFolder)
		if err != nil {
			newErr := errors.Join(
				errors.New("could not write route \""+routes[i].Name+"\" to file"),
				err,
			)
			errs = append(errs, newErr)
			continue
		}
	}

	return errs
}

/*
Find a route in a .gpx file, then create a .route.json file for that route.
Returns an error if the named route is not in the .gpx file.
*/
func FindAndCreateRoute(routeName string, inputGpxFilePath, outputFolder string) error {
	gpxFile, err := gpx.ParseFile(inputGpxFilePath)
	if err != nil {
		return errors.Join(errors.New("error parsing .gpx file"), err)
	}

	var gpxRoute *gpx.GPXRoute = nil
	for _, route := range gpxFile.Routes {
		if routeName == route.Name {
			gpxRoute = &route
			break
		}
	}
	if gpxRoute == nil {
		return errors.New("route not found in .gpx file")
	}

	createdRoute, err := createRouteFromGpx(gpxRoute)
	if err != nil {
		return errors.Join(
			errors.New("could not get route data for route \""+gpxRoute.Name+"\""),
			err,
		)
	}

	err = writeRouteToFile(createdRoute, outputFolder)
	if err != nil {
		return errors.Join(
			errors.New("could not write route \""+createdRoute.Name+"\" to file"),
			err,
		)
	}

	return nil
}

/*
Loads a route from a .json file stored on the user’s computer.
The route files must be generated separately with CreateRoutes() or FindAndCreateRoute().
*/
func LoadRoute(routeFilePath string) (*types.Route, error) {
	functionErrMsg := errors.New("error reading route file")

	file, err := os.Open(routeFilePath)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}

	defer file.Close()

	var route types.Route

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&route)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}

	for i, section := range route.Sections {
		if i >= len(route.Sections)-1 {
			break
		}
		section.Next = &route.Sections[i+1]
	}

	return &route, nil
}

// Finds the section in the given route closest to the given coordinates.
// EXPORT LATER - CURRENTLY UNSUPPORTED
func findSectionFromCoordinates(
	route *types.Route,
	coordinates types.Coordinates,
) (*types.RouteSection, error) {
	// TODO: implement
	return nil, nil
}

// Finds the section in the given route closest to the given address.
// EXPORT LATER - CURRENTLY UNSUPPORTED
func findSectionFromAddress(
	route *types.Route,
	address string,
) (*types.RouteSection, error) {
	// TODO: implement
	return nil, nil
}

// const maxRoutepointsPerRequest int = 860 // May need to set this through configs in the future
const miToFt float64 = 5280.0
const mToFt float64 = 3.28084
const hoursToSeconds float64 = 3600.0

func createRouteFromGpx(gpxRoute *gpx.GPXRoute) (*types.Route, error) {
	functionErrMsg := errors.New("error creating route")
	var route types.Route

	route.Name = gpxRoute.Name
	if route.Name == "" {
		route.Name = "myroute"
	}

	// True if the route name ends in "loop"
	// Add option to set this manually?
	route.IsLoop = len(gpxRoute.Name) >= 4 &&
		gpxRoute.Name[len(gpxRoute.Name)-4:] == "Loop"

	// Parse routepoints and get data
	coordinates := make([]types.Coordinates, len(gpxRoute.Points))
	for i, point := range gpxRoute.Points {
		coordinates[i].Latitude = point.Latitude
		coordinates[i].Longitude = point.Longitude
	}

	prevCoordinates := coordinates[0]
	prevElevation := gpxRoute.Points[0].Elevation.Value() * mToFt

	// ORS only accepts 50 waypoints per request, so we have to break the request up
	for currIndex := 0; currIndex < len(coordinates); currIndex += 50 {
		sliceEnd := int(math.Min(
			float64(len(coordinates)),
			float64(currIndex+50),
		))

		directions, err := ors.GetDirections(coordinates[currIndex:sliceEnd])
		if err != nil {
			return nil, errors.Join(functionErrMsg, err)
		}
		if currIndex == 0 && len(directions.Routes) <= 0 {
			return nil, errors.Join(
				functionErrMsg,
				errors.New("no routes returned from OpenRouteService"),
			)
		}

		for _, segment := range directions.Routes[0].Segments {
			for _, step := range segment.Steps {
				// "Goal" means that you've arrived at your destination.
				// Since the input GPX file has hundreds of intermediate points, we will
				// get hundreds of "you have arrived" steps from ORS, so we need to remove them.
				if step.InstructionType == int(types.Goal) || step.Distance == 0 {
					continue
				}

				var section types.RouteSection

				section.LengthFt = step.Distance * miToFt

				section.CoordinatesInitial = prevCoordinates
				if step.Maneuver.Location != nil {
					section.CoordinatesFinal = *step.Maneuver.Location
				} else {
					section.CoordinatesFinal = coordinates[len(coordinates)-1]
				}

				section.ElevationInitialFt = prevElevation
				section.ElevationFinalFt = findClosestGpxPointToCoordinates(
					section.CoordinatesFinal,
					gpxRoute.Points,
				).Elevation.Value() * mToFt

				section.ExitInstruction = step.Instruction
				section.InstructionCode = types.RouteInstruction(step.InstructionType)

				// Not actually the speed limit - currently calculates average speed
				// of this step based on the distance and duration.
				section.SpeedLimitMph = uint(step.Distance / (step.Duration / hoursToSeconds))

				section.PositionInRoute = len(route.Sections)
				section.Next = nil

				var prevSection *types.RouteSection = nil
				if length := len(route.Sections); length > 0 {
					prevSection = &route.Sections[length-1]
				}

				if prevSection != nil &&
					prevSection.InstructionCode == types.Depart &&
					section.InstructionCode == types.Depart &&
					prevSection.ExitInstruction == section.ExitInstruction {
					// ORS creates a "Depart from this location" instruction for every waypoint you give it as input.
					// Since we always send multiple waypoints in each request, this results in a lot of duplicate
					// "Depart" instructions. This ugly code merges them.
					*prevSection = mergeRouteSections(prevSection, &section)
				} else {
					route.Sections = append(route.Sections, section)

					if prevSection != nil {
						prevSection.Next = &route.Sections[len(route.Sections)-1]
					}
				}

				prevCoordinates = section.CoordinatesFinal
				prevElevation = section.ElevationFinalFt
			}
		}

		if currIndex+50 < len(coordinates) {
			// Wait 1.5 seconds to get around ORS API rate limits
			time.Sleep(1500 * time.Millisecond)
		}

	}

	return &route, nil
}

// Simple linear search using pythagorean distance.
// Could this be improved with a binary search?
func findClosestGpxPointToCoordinates(coordinates types.Coordinates, points []gpx.GPXPoint) *gpx.GPXPoint {
	minDistance := math.MaxFloat64
	var closestPoint *gpx.GPXPoint = nil

	for i := range points {
		latitudeDiff := coordinates.Latitude - points[i].Latitude
		longitudeDiff := coordinates.Longitude - points[i].Longitude

		distance := math.Sqrt(math.Pow(latitudeDiff, 2) + math.Pow(longitudeDiff, 2))

		if distance < minDistance {
			minDistance = distance
			closestPoint = &points[i]
		}
	}

	return closestPoint
}

/*
Appends the second route section into the first.
Should only be used in createRouteFromGpx()
*/
func mergeRouteSections(first *types.RouteSection, second *types.RouteSection) types.RouteSection {
	var combined types.RouteSection

	combined.LengthFt = first.LengthFt + second.LengthFt

	combined.CoordinatesInitial = first.CoordinatesInitial
	combined.CoordinatesFinal = second.CoordinatesFinal

	combined.ElevationInitialFt = first.ElevationInitialFt
	combined.ElevationFinalFt = second.ElevationFinalFt

	combined.ExitInstruction = first.ExitInstruction
	combined.InstructionCode = first.InstructionCode

	combined.PositionInRoute = first.PositionInRoute
	combined.Next = second.Next

	// Weighted average of both speeds
	combined.SpeedLimitMph = uint(
		float64(first.SpeedLimitMph)*(first.LengthFt/combined.LengthFt) +
			float64(second.SpeedLimitMph)*(second.LengthFt/combined.LengthFt),
	)

	return combined
}

func writeRouteToFile(route *types.Route, outputFolder string) error {
	functionErrMsg := errors.New("error writing route to file")

	// Remove slash from last character of output path
	lastChar := len(outputFolder) - 1
	if lastChar > 0 && (outputFolder[lastChar] == '/' || outputFolder[lastChar] == '\\') {
		outputFolder = outputFolder[:lastChar]
	}

	filePath := outputFolder + "/" + route.Name + ".route.json"
	file, err := os.Create(filePath)
	if err != nil {
		return errors.Join(functionErrMsg, err)
	}

	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(*route)
	if err != nil {
		return errors.Join(functionErrMsg, err)
	}

	return nil
}
