package ors

import (
	"asc-simulation/types"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

func GetDirections(coordinates []types.Coordinates) (*DirectionsResponse, error) {
	functionErrMsg := errors.New("error getting directions from OpenRouteService")

	formattedCoordinates := make([][]float64, len(coordinates))
	for i, coordinate := range coordinates {
		formattedCoordinates[i] = encodeCoordinates(coordinate)
	}

	requestBody := DirectionsRequest{
		Coordinates: formattedCoordinates,
		Attributes:  []string{"avgspeed"},
		Elevation:   true,
		Maneuvers:   true,
		Units:       "mi",
	}
	requestBodyJson, _ := json.Marshal(requestBody)

	orsUrl := os.Getenv("OPEN_ROUTE_SERVICE_URL")
	if orsUrl == "" {
		return nil, errors.Join(
			functionErrMsg,
			errors.New("no URL found for OpenRouteService"),
		)
	}

	orsToken := os.Getenv("OPEN_ROUTE_SERVICE_TOKEN")
	if orsToken == "" {
		return nil, errors.Join(
			functionErrMsg,
			errors.New("no authorization token found for OpenRouteService"),
		)
	}

	requestPath := "v2/directions/driving-car"

	request, err := http.NewRequest(
		http.MethodPost,
		orsUrl+requestPath,
		bytes.NewReader(requestBodyJson),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", orsToken)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}
	if response.StatusCode >= 400 {
		respBody, _ := io.ReadAll(response.Body)
		fmt.Println("\nResponse: ", string(respBody), "\n")

		return nil, errors.Join(
			functionErrMsg,
			errors.New("request to OpenRouteService failed"),
			errors.New("response status: "+response.Status),
		)
	}

	var result DirectionsResponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&result)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}

	return &result, nil
}
