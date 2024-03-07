package dataaccess

import "asc-simulation/types"

// Loads live traffic data at the specified section of the route.
func GetTraffic(section types.RouteSection, options TrafficDataOptions) (*types.Traffic, error) {
	return nil, nil // TODO: implement
}

/*
This struct exists so we change the arguments to GetWeather() without having to
change the code everywhere GetTraffic() is used. This also allows us to use default values.
*/
type TrafficDataOptions struct {
	// Amount of time to wait before fetching new traffic data from API
	RefreshRateSeconds float64
}
