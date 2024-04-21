package solcast

import (
	"asc-simulation/types"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

// Currently used for both live and forecasted weather data
var outputParameters = []string{
	"air_temp",
	"cloud_opacity",
	"dewpoint_temp",
	"precipitation_rate",
	"surface_pressure",
	"wind_direction_10m",
	"wind_speed_10m",
	"zenith",
}

func GetLiveIrradianceAndWeather(
	coordinates types.Coordinates,
	responseWindowHours int,
	responseIntervalMinutes TimePeriod,
) (*LiveIrradianceAndWeatherResponse, error) {
	functionErrMsg := errors.New("error getting weather from Solcast")

	solcastUrl := os.Getenv("SOLCAST_URL")
	if solcastUrl == "" {
		return nil, errors.Join(
			functionErrMsg,
			errors.New("no URL found for Solcast API"),
		)
	}

	solcastToken := os.Getenv("SOLCAST_TOKEN")
	if solcastToken == "" {
		return nil, errors.Join(
			functionErrMsg,
			errors.New("no authorization token found for Solcast API"),
		)
	}

	requestPath := "live/radiation_and_weather"
	request, err := http.NewRequest(
		http.MethodGet,
		solcastUrl+requestPath,
		bytes.NewReader(nil),
	)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}

	request.Header.Set("Authorization", "Bearer "+solcastToken)

	query := request.URL.Query()
	query.Add("latitude", fmt.Sprint(coordinates.Latitude))
	query.Add("longitude", fmt.Sprint(coordinates.Longitude))
	query.Add("hours", fmt.Sprint(responseWindowHours))
	query.Add("period", string(responseIntervalMinutes))

	for _, param := range outputParameters {
		query.Add("output_parameters", param)
	}

	query.Add("format", "json")
	request.URL.RawQuery = query.Encode()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}
	if response.StatusCode >= 400 {
		return nil, errors.Join(
			functionErrMsg,
			errors.New("request to Solcast failed"),
		)
	}

	var result LiveIrradianceAndWeatherResponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&result)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}

	if len(result.EstimatedActuals) <= 0 {
		return nil, errors.New("no weather data returned from Solcast")
	}

	return &result, nil
}

func GetForecastIrradianceAndWeather(
	coordinates types.Coordinates,
	responseWindowHours int,
	responseIntervalMinutes TimePeriod,
) (*ForecastIrradianceAndWeatherResponse, error) {
	functionErrMsg := errors.New("error getting weather from Solcast")

	solcastUrl := os.Getenv("SOLCAST_URL")
	if solcastUrl == "" {
		return nil, errors.Join(
			functionErrMsg,
			errors.New("no URL found for Solcast API"),
		)
	}

	solcastToken := os.Getenv("SOLCAST_TOKEN")
	if solcastToken == "" {
		return nil, errors.Join(
			functionErrMsg,
			errors.New("no authorization token found for Solcast API"),
		)
	}

	requestPath := "forecast/radiation_and_weather"
	request, err := http.NewRequest(
		http.MethodGet,
		solcastUrl+requestPath,
		bytes.NewReader(nil),
	)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}

	request.Header.Set("Authorization", "Bearer "+solcastToken)

	query := request.URL.Query()
	query.Add("latitude", fmt.Sprint(coordinates.Latitude))
	query.Add("longitude", fmt.Sprint(coordinates.Longitude))
	query.Add("hours", fmt.Sprint(responseWindowHours))
	query.Add("period", string(responseIntervalMinutes))

	for _, param := range outputParameters {
		query.Add("output_parameters", param)
	}

	query.Add("format", "json")
	request.URL.RawQuery = query.Encode()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}
	if response.StatusCode >= 400 {
		return nil, errors.Join(
			functionErrMsg,
			errors.New("request to Solcast failed"),
		)
	}

	var result ForecastIrradianceAndWeatherResponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&result)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}

	if len(result.Forecasts) <= 0 {
		return nil, errors.New("no weather data returned from Solcast")
	}

	return &result, nil
}
