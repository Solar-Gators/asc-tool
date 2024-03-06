package dataaccess

import "asc-simulation/types"

/*
Creates a route file (which is just a .json file) from a .gpx file.
Parses coordinate and elevation data from the .gpx file, calls third-party APIs to get
other data, and writes all of that data to a route file.
This should not be called during a simulation!
*/
func CreateRoute(inputGpxFilePath string, outputRouteFilePath string) error {
	// Parse GPX file
	// Call APIs to get supplementary data
	// Feed coordinate and elevation data into RouteSections struct
	// Put data in RouteSections struct
	// Convert data to json
	// Write to output file

	// TODO: Implement
	return nil
}

/*
Loads a route from a .json file stored on the user’s computer.
The route files must be generated separately with CreateRoute().
*/
func LoadRoute(routeFilePath string) (*types.Route, error) {
	// TODO: implement
	return nil, nil
}

// Finds the section in the given route closest to the given coordinates.
func FindSectionFromCoordinates(
	route *types.Route,
	coordinates types.Coordinates,
) (*types.RouteSection, error) {
	// TODO: implement
	return nil, nil
}

// Finds the section in the given route closest to the given address.
func FindSectionFromAddress(
	route *types.Route,
	address string,
) (*types.RouteSection, error) {
	// TODO: implement
	return nil, nil
}
