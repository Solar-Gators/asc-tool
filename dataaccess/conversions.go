package dataaccess

const miToFt float64 = 5280.0
const mToFt float64 = 3.28084
const ftToIn float64 = 12.0
const hoursToSeconds float64 = 3600.0
const hPaToPsi float64 = 0.0145038

func celsiusToFahrenheit(temperature float64) float64 {
	return (temperature * (9.0 / 5.0)) + 32.0
}

func msToMph(speedMetersPerSecond float64) float64 {
	return (speedMetersPerSecond * (mToFt / miToFt)) / hoursToSeconds
}
