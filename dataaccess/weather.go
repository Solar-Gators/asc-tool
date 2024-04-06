package dataaccess

import "asc-simulation/types"

// Loads live weather data at the specified section of the route.
func GetWeather(section types.RouteSection, options WeatherDataOptions) (*types.Weather, error) {
	return nil, nil // TODO: implement
}

/*
This struct exists so we change the arguments to GetWeather() without having to
change the code everywhere GetWeather() is used. This also allows us to use default values.
*/
type WeatherDataOptions struct {
	// Amount of time to wait before fetching new weather data from API
	RefreshRateSeconds float64
}
