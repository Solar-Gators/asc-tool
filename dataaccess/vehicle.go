package dataaccess

import (
	"asc-simulation/types"
	"encoding/json"
	"errors"
	"os"
)

// Loads vehicle data from a .json file provided by the user.
func GetVehicle(vehicleFilePath string) (*types.Vehicle, error) {
	functionErrMsg := errors.New("error loading vehicle data")

	file, err := os.Open(vehicleFilePath)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}
	defer file.Close()

	var vehicle types.Vehicle

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&vehicle)
	if err != nil {
		return nil, errors.Join(functionErrMsg, err)
	}

	return &vehicle, nil
}

// Saves updated vehicle data to a .json file.
func UpdateVehicle(vehicleFilePath string, vehicle *types.Vehicle) error {
	functionErrMsg := errors.New("error saving vehicle data")

	file, err := os.Create(vehicleFilePath)
	if err != nil {
		return errors.Join(functionErrMsg, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(*vehicle)
	if err != nil {
		return errors.Join(functionErrMsg, err)
	}

	return nil
}
