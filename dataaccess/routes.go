package dataaccess

import "asc-simulation/types"

/*
Creates route files (which is just a .json file) from a .gpx file.
Each route in the .gpx file will be written to a separate .json file in
the output folder.

This function should not be called during a simulation!
It calls external APIs to get additional route data.
*/
func CreateRoutes(inputGpxFilePath string, outputFolder string) error {
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
