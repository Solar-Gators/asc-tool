package solcast

// Enum to represent fixed time periods in the Solcast API.
type TimePeriod string

const (
	Period5Mins  TimePeriod = "PT5M"
	Period10Mins TimePeriod = "PT10M"
	Period15Mins TimePeriod = "PT15M"
	Period20Mins TimePeriod = "PT20M"
	Period30Mins TimePeriod = "PT30M"
	Period60Mins TimePeriod = "PT60M"
)

func (period *TimePeriod) MinuteValue() int {
	switch *period {
	case Period5Mins:
		return 5
	case Period10Mins:
		return 10
	case Period15Mins:
		return 15
	case Period20Mins:
		return 20
	case Period30Mins:
		return 30
	case Period60Mins:
		return 60
	}

	return 0
}

type LiveIrradianceAndWeatherActuals struct {
	AirTemp           float64 `json:"air_temp"`
	CloudOpacity      float64 `json:"cloud_opacity"`
	DewpointTemp      float64 `json:"dewpoint_temp"`
	PrecipitationRate float64 `json:"precipitation_rate"`
	SurfacePressure   float64 `json:"surface_pressure"`
	WindDirection10m  float64 `json:"wind_direction_10m"`
	WindSpeed10m      float64 `json:"wind_speed_10m"`
	Zenith            float64
}

type LiveIrradianceAndWeatherResponse struct {
	EstimatedActuals []LiveIrradianceAndWeatherActuals `json:"estimated_actuals"`
}

// Should make this into a separate struct if we ever need to get data
// from the forecast that isn't in the live weather data
type ForecastIrradianceAndWeather LiveIrradianceAndWeatherActuals

type ForecastIrradianceAndWeatherResponse struct {
	Forecasts []ForecastIrradianceAndWeather
}
