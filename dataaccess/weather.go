package dataaccess

import (
	"asc-simulation/dataaccess/solcast"
	"asc-simulation/types"
	"errors"
	"math"
	"strconv"
	"time"
)

/*
Loads live weather data at the specified section of the route.
Call DefaultWeatherOptions() to load the default weather options.
*/
func GetWeather(section *types.RouteSection, options WeatherDataOptions) (*types.Weather, error) {
	cacheSection := getWeatherCacheSection(section, &options)
	if cacheSection != nil && cacheSection.weather != nil {
		return cacheSection.weather, nil
	}

	responseInterval := solcast.Period5Mins
	solcastResponse, err := solcast.GetLiveIrradianceAndWeather(
		section.CoordinatesInitial,
		1,
		responseInterval,
	)
	if err != nil {
		return nil, errors.Join(errors.New("error getting weather data"), err)
	}

	evaporationRateInputs := make([]evaporationRateInput, len(solcastResponse.EstimatedActuals))
	precipitationRates := make([]float64, len(solcastResponse.EstimatedActuals))
	for i, weatherData := range solcastResponse.EstimatedActuals {
		evaporationRateInputs[i] = evaporationRateInput{
			windSpeedMpS:   weatherData.WindSpeed10m,
			airPressurehPa: weatherData.SurfacePressure,
			airTempC:       weatherData.AirTemp,
			dewPointC:      weatherData.DewpointTemp,
		}
		precipitationRates[i] = weatherData.PrecipitationRate
	}

	mostRecentWeather := solcastResponse.EstimatedActuals[0]
	weather := types.Weather{
		SolarZenithDegrees:   mostRecentWeather.Zenith,
		AirTempDegreesF:      celsiusToFahrenheit(mostRecentWeather.AirTemp),
		CloudCoverPercentage: mostRecentWeather.CloudOpacity,
		WindSpeedMph:         msToMph(mostRecentWeather.WindSpeed10m),
		WindDirectionDegrees: mostRecentWeather.WindDirection10m,
		RainOnGroundInches:   accumulatedRainfallInches(evaporationRateInputs, precipitationRates, responseInterval),
	}

	if options.UsingWeatherCache {
		if cacheSection == nil {
			key := section.Route.Name
			weatherCache[key] = createWeatherCacheSectionsForRoute(section.Route)
			cacheSection = getWeatherCacheSection(section, &options)
		}

		now := time.Now()
		cacheSection.weather = &weather
		cacheSection.collectedAt = now
		cacheSection.occursAt = now
	}

	return &weather, nil
}

/*
Loads forecasted weather data at the specified section of the route, at the specified time.
Can get forecasted data up to 336 hours (14 days) in the future.
Call DefaultWeatherOptions() to load the default weather options.
*/
func GetWeatherForecast(
	section *types.RouteSection,
	hoursInFuture int,
	options WeatherDataOptions,
) (*types.Weather, error) {
	if hoursInFuture > 336 {
		return nil, errors.New("cannot get forecast more than 336 hours in the future")
	}

	cacheSection := getWeatherCacheSection(section, &options)
	if cacheSection.weather != nil {
		return cacheSection.weather, nil
	}

	responseInterval := solcast.Period5Mins

	solcastResponse, err := solcast.GetForecastIrradianceAndWeather(
		section.CoordinatesInitial,
		hoursInFuture,
		responseInterval,
	)
	if err != nil {
		return nil, errors.Join(errors.New("error getting weather data"), err)
	}

	evaporationRateInputs := make([]evaporationRateInput, len(solcastResponse.Forecasts))
	precipitationRates := make([]float64, len(solcastResponse.Forecasts))
	for i, weatherData := range solcastResponse.Forecasts {
		evaporationRateInputs[i] = evaporationRateInput{
			windSpeedMpS:   weatherData.WindSpeed10m,
			airPressurehPa: weatherData.SurfacePressure,
			airTempC:       weatherData.AirTemp,
			dewPointC:      weatherData.DewpointTemp,
		}
		precipitationRates[i] = weatherData.PrecipitationRate
	}

	weatherAtTargetTime := solcastResponse.Forecasts[len(solcastResponse.Forecasts)-1]
	weather := types.Weather{
		SolarZenithDegrees:   weatherAtTargetTime.Zenith,
		AirTempDegreesF:      celsiusToFahrenheit(weatherAtTargetTime.AirTemp),
		CloudCoverPercentage: weatherAtTargetTime.CloudOpacity,
		WindSpeedMph:         msToMph(weatherAtTargetTime.WindSpeed10m),
		WindDirectionDegrees: weatherAtTargetTime.WindDirection10m,
		RainOnGroundInches:   accumulatedRainfallInches(evaporationRateInputs, precipitationRates, responseInterval),
	}

	if options.UsingWeatherCache {
		if cacheSection == nil {
			key := section.Route.Name
			weatherCache[key] = createWeatherCacheSectionsForRoute(section.Route)
			cacheSection = getWeatherCacheSection(section, &options)
		}

		now := time.Now()
		cacheSection.weather = &weather
		cacheSection.collectedAt = now
		cacheSection.occursAt = now.Add(time.Duration(hoursInFuture) * time.Hour)
	}

	return &weather, nil
}

/*
Loads forecasted weather data at the specified section of the route, at the specified time.
Can get forecasted data up to 336 hours (14 days) in the future.
Call DefaultWeatherOptions() to load the default weather options.
*/
func GetWeatherForecastAtTime(
	section *types.RouteSection,
	targetTime time.Time,
	options WeatherDataOptions,
) (*types.Weather, error) {
	timeUntilTarget := time.Until(targetTime)
	return GetWeatherForecast(section, int(timeUntilTarget.Hours()), options)
}

/*
This struct exists so we change the arguments to GetWeather() without having to
change the code everywhere GetWeather() is used. This also allows us to use default values.
*/
type WeatherDataOptions struct {
	// If true, the same weather data will be returned for locations that are close to each other.
	// This must be must be false to force a new API call. True by default.
	UsingWeatherCache bool
	// Amount of time to wait before fetching new weather data from API
	RefreshTimeSeconds float64
}

func DefaultWeatherDataOptions() WeatherDataOptions {
	return WeatherDataOptions{
		UsingWeatherCache:  true,
		RefreshTimeSeconds: 1 * hoursToSeconds,
	}
}

type weatherCacheSection struct {
	combinedLengthFt  float64
	occursAt          time.Time
	collectedAt       time.Time
	startSectionIndex int
	endSectionIndex   int
	weather           *types.Weather
}

const weatherCacheSectionMinLengthFt float64 = 2 * miToFt

var weatherCache = make(map[string][]weatherCacheSection)

func getWeatherCacheSection(section *types.RouteSection, options *WeatherDataOptions) *weatherCacheSection {
	if !options.UsingWeatherCache {
		return nil
	}

	key := section.Route.Name
	if key == "" {
		key = createDefaultCacheKey(section)
	}

	weatherCacheSections, exists := weatherCache[key]
	if !exists {
		return nil
	}

	// Find the corresponding section using binary search
	var cacheSection *weatherCacheSection = nil
	left, right := 0, len(weatherCacheSections)-1
	for left <= right {
		mid := (left + right) / 2
		currSection := &weatherCacheSections[mid]

		if section.PositionInRoute >= currSection.startSectionIndex &&
			section.PositionInRoute <= currSection.endSectionIndex {

			cacheSection = currSection
			break
		} else if section.PositionInRoute < currSection.startSectionIndex {
			right = mid
		} else if section.PositionInRoute > currSection.endSectionIndex {
			left = mid
		}
	}
	if cacheSection == nil {
		return nil
	}

	duration := time.Duration(options.RefreshTimeSeconds * float64(time.Second))
	expirationTime := cacheSection.collectedAt.Add(duration)

	if time.Now().After(expirationTime) {
		cacheSection.weather = nil
	}

	return cacheSection
}

func createDefaultCacheKey(section *types.RouteSection) string {
	// If the route does not have a name, then the
	// cache key is coordinates + exit instruction + section length of a route's
	// first segment. Not guaranteed to be unique but highly unlikely we'd ever
	// have a collision.
	return strconv.FormatFloat(section.CoordinatesInitial.Latitude, 'f', 6, 64) +
		"," +
		strconv.FormatFloat(section.CoordinatesInitial.Longitude, 'f', 6, 64) +
		" " +
		section.ExitInstruction +
		strconv.FormatFloat(section.LengthFt, 'f', -1, 64)
}

func createWeatherCacheSectionsForRoute(route *types.Route) []weatherCacheSection {
	cacheSections := make([]weatherCacheSection, 0)

	for _, section := range route.Sections {
		lastIndex := len(cacheSections) - 1

		if section.PositionInRoute > 0 &&
			cacheSections[lastIndex].combinedLengthFt < weatherCacheSectionMinLengthFt {
			// Merge RouteSection into previous WeatherCacheSection
			cacheSections[lastIndex].combinedLengthFt += section.LengthFt
			cacheSections[lastIndex].endSectionIndex = section.PositionInRoute
		} else {
			cacheSections = append(cacheSections, weatherCacheSection{
				combinedLengthFt:  section.LengthFt,
				startSectionIndex: section.PositionInRoute,
				endSectionIndex:   section.PositionInRoute,
				weather:           nil,
			})
		}
	}

	return cacheSections
}

type evaporationRateInput struct {
	windSpeedMpS   float64
	airPressurehPa float64
	airTempC       float64
	dewPointC      float64
}

// This is an estimate - see comment for rainEvaporationRateInchesPerHour()
func accumulatedRainfallInches(
	evaporationRateInputs []evaporationRateInput,
	precipitationRatesMmph []float64,
	responseTimeInterval solcast.TimePeriod,
) float64 {
	accumulatedRain := 0.0
	minuteRatio := float64(responseTimeInterval.MinuteValue()) / 60.0

	for i := len(evaporationRateInputs) - 1; i >= 0; i-- {
		accumulatedRain += (precipitationRatesMmph[i] / 1000) * mToFt * ftToIn * minuteRatio
		accumulatedRain -= rainEvaporationRateInchesPerHour(evaporationRateInputs[i]) * minuteRatio
		accumulatedRain = math.Max(0, accumulatedRain)
	}

	return accumulatedRain
}

/*
THIS IS AN ESTIMATE!
This estimates rain evaporation per hour, but it's based off an equation meant for
evaporation of water over other bodies of water (like puddles or pools).

Right now, searching for an equation specifically for roads returns a bunch of research
papers with inconsistent findings, or equations that require data we can't get from Solcast.
So, this equation may not be 100% accurate, and it may be a good idea to use another equation
in the future.

Source: https://www.engineeringtoolbox.com/evaporation-water-surface-d_690.html
*/
func rainEvaporationRateInchesPerHour(input evaporationRateInput) float64 {
	evaporationCoefficient := 25.0 + 19.0*input.windSpeedMpS
	surfaceAreaSquareMeters := 1.0 // Can be any value, gets divided out at the end

	currVaporPressurehPa := vaporPressurehPa(input.dewPointC)
	saturationVaporPressurehPa := vaporPressurehPa(input.airTempC)

	currHumidityRatio := humidityRatio(input.airPressurehPa, currVaporPressurehPa)
	saturationHumidityRatio := humidityRatio(input.airPressurehPa, saturationVaporPressurehPa)

	evaporatedRainMassKg := evaporationCoefficient *
		surfaceAreaSquareMeters *
		(saturationHumidityRatio - currHumidityRatio)

	const waterDensity float64 = 997.0
	rainfallMeters := evaporatedRainMassKg / (waterDensity * surfaceAreaSquareMeters)

	return rainfallMeters * mToFt * ftToIn
}

func humidityRatio(airPressurehPa float64, vaporPressurehPa float64) float64 {
	return 0.62198 * vaporPressurehPa / (airPressurehPa - vaporPressurehPa)
}

// Source: https://www.weather.gov/media/epz/wxcalc/vaporPressure.pdf
func vaporPressurehPa(airTempC float64) float64 {
	return 6.11 * 10 * (7.5 * airTempC / (237.3 + airTempC))
}
