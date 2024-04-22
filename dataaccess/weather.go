package dataaccess

import (
	"asc-simulation/dataaccess/solcast"
	"asc-simulation/types"
	"encoding/json"
	"errors"
	"math"
	"os"
	"strconv"
	"time"
)

/*
Loads live weather data at the specified section of the route.
Call DefaultWeatherOptions() to load the default weather options.
*/
func GetWeather(section *types.RouteSection, options WeatherDataOptions) (*types.Weather, error) {
	cacheSection, err := getWeatherCacheSection(section, &options)
	if err != nil {
		// If the JSON Weather cache fails, we do NOT want to make calls to Solcast anyway.
		// If that happens during an optimization run, the API endpoints could be called
		// hundreds/thousands of times extra, using up our available API calls.
		return nil, err
	}
	if cacheSection != nil && cacheSection.Weather != nil {
		return cacheSection.Weather, nil
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
		SurfacePressurePsi:   mostRecentWeather.SurfacePressure * hPaToPsi,
	}

	if options.UsingWeatherCache {
		key := section.Route.Name

		if cacheSection == nil {
			weatherCache[key] = createWeatherCacheSectionsForRoute(section.Route)
			cacheSection, _ = getWeatherCacheSection(section, &options)
		}

		now := time.Now()
		cacheSection.Weather = &weather
		cacheSection.CollectedAt = now
		cacheSection.OccursAt = now

		err = saveWeatherCacheToJson(key, weatherCache[key])
		if err != nil {
			return &weather, err
		}
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

	cacheSection, err := getWeatherCacheSection(section, &options)
	if err != nil {
		// If the JSON Weather cache fails, we do NOT want to make calls to Solcast anyway.
		// If that happens during an optimization run, the API endpoints could be called
		// hundreds/thousands of times extra, using up our available API calls.
		return nil, err
	}
	if cacheSection != nil && cacheSection.Weather != nil {
		return cacheSection.Weather, nil
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
		SurfacePressurePsi:   weatherAtTargetTime.SurfacePressure * hPaToPsi,
	}

	if options.UsingWeatherCache {
		key := section.Route.Name

		if cacheSection == nil {
			weatherCache[key] = createWeatherCacheSectionsForRoute(section.Route)
			cacheSection, _ = getWeatherCacheSection(section, &options)
		}

		now := time.Now()
		cacheSection.Weather = &weather
		cacheSection.CollectedAt = now
		cacheSection.OccursAt = now.Add(time.Duration(hoursInFuture) * time.Hour)

		err = saveWeatherCacheToJson(key, weatherCache[key])
		if err != nil {
			return &weather, err
		}
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
	CombinedLengthFt  float64
	OccursAt          time.Time
	CollectedAt       time.Time
	StartSectionIndex int
	EndSectionIndex   int
	Weather           *types.Weather
}

const weatherCacheSectionMinLengthFt float64 = 2 * miToFt

var weatherCache = make(map[string][]weatherCacheSection)

func getWeatherCacheSection(section *types.RouteSection, options *WeatherDataOptions) (*weatherCacheSection, error) {
	if !options.UsingWeatherCache {
		return nil, nil
	}

	key := section.Route.Name
	if key == "" {
		key = createDefaultCacheKey(section)
	}

	weatherCacheSections, existsInMemory := weatherCache[key]
	if !existsInMemory {
		savedCacheSection, err := readWeatherCacheFromJson(key)
		if err != nil {
			return nil, err
		}
		if savedCacheSection == nil {
			return nil, nil
		}

		weatherCache[key] = savedCacheSection
		weatherCacheSections = weatherCache[key]
	}

	// Find the corresponding section using binary search
	var cacheSection *weatherCacheSection = nil
	left, right := 0, len(weatherCacheSections)-1
	for left <= right {
		mid := (left + right) / 2
		currSection := &weatherCacheSections[mid]

		if section.PositionInRoute >= currSection.StartSectionIndex &&
			section.PositionInRoute <= currSection.EndSectionIndex {

			cacheSection = currSection
			break
		} else if section.PositionInRoute < currSection.StartSectionIndex {
			right = mid - 1
		} else if section.PositionInRoute > currSection.EndSectionIndex {
			left = mid + 1
		}
	}
	if cacheSection == nil {
		return nil, nil
	}

	duration := time.Duration(options.RefreshTimeSeconds * float64(time.Second))
	expirationTime := cacheSection.CollectedAt.Add(duration)

	if time.Now().After(expirationTime) {
		cacheSection.Weather = nil
	}

	return cacheSection, nil
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
			cacheSections[lastIndex].CombinedLengthFt < weatherCacheSectionMinLengthFt {
			// Merge RouteSection into previous WeatherCacheSection
			cacheSections[lastIndex].CombinedLengthFt += section.LengthFt
			cacheSections[lastIndex].EndSectionIndex = section.PositionInRoute
		} else {
			cacheSections = append(cacheSections, weatherCacheSection{
				CombinedLengthFt:  section.LengthFt,
				StartSectionIndex: section.PositionInRoute,
				EndSectionIndex:   section.PositionInRoute,
				Weather:           nil,
			})
		}
	}

	return cacheSections
}

func readWeatherCacheFromJson(cacheKey string) ([]weatherCacheSection, error) {
	functionErrMsg := errors.New("error loading weather from json cache")

	filePath := getWeatherCacheFilePath(cacheKey)
	file, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, errors.Join(functionErrMsg, err)
	}
	defer file.Close()

	var cacheSections []weatherCacheSection

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cacheSections)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}

	return cacheSections, nil
}

func saveWeatherCacheToJson(cacheKey string, cacheSections []weatherCacheSection) error {
	functionErrMsg := errors.New("error saving weather to json cache")

	outputFolder := getWeatherCacheOutputFolder()
	err := os.MkdirAll(outputFolder, os.ModePerm)
	if err != nil {
		return errors.Join(functionErrMsg, err)
	}

	filePath := getWeatherCacheFilePath(cacheKey)
	file, err := os.Create(filePath)
	if err != nil {
		return errors.Join(functionErrMsg, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(cacheSections)
	if err != nil {
		return errors.Join(functionErrMsg, err)
	}

	return nil
}

func getWeatherCacheFilePath(cacheKey string) string {
	outputFolder := getWeatherCacheOutputFolder()

	fileName := removeIllegalFilenameChars(cacheKey)
	filePath := outputFolder + "/" + fileName + ".json"

	return filePath
}

func getWeatherCacheOutputFolder() string {
	outputFolder, _ := os.UserCacheDir()

	// Remove slash from last character of output path
	lastChar := len(outputFolder) - 1
	if lastChar > 0 && (outputFolder[lastChar] == '/' || outputFolder[lastChar] == '\\') {
		outputFolder = outputFolder[:lastChar]
	}
	outputFolder += "/asc-tool/weather"

	return outputFolder
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
