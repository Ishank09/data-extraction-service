package env

import (
	"encoding/base64"
	"os"
	"strconv"

	"github.com/ivasania/data-extraction-service/pkg/logging"
)

var log = logging.GetLogger()

func GetOrDie(envVar string) string {
	if os.Getenv(envVar) == "" {
		log.Fatal("Environment variable %s is not set", envVar)
	}

	return os.Getenv(envVar)
}

func GetB64EncodedEnvOrDefault(envVar string, defaultValue []byte) []byte {
	if os.Getenv(envVar) == "" {
		return defaultValue
	}

	return Bas64DecodeOrDie(os.Getenv(envVar))
}
func GetOrDefault(envVar, defaultValue string) string {
	if os.Getenv(envVar) == "" {
		return defaultValue
	}

	return os.Getenv(envVar)
}

func GetOrDefaultBool(envVar string, defaultValue bool) bool {
	if os.Getenv(envVar) == "" {
		return defaultValue
	}

	return os.Getenv(envVar) == "true"
}

func ParseInt(envVar string, defaultValue int64) int64 {
	if os.Getenv(envVar) == "" {
		return defaultValue
	}

	result, err := strconv.ParseInt(os.Getenv(envVar), 10, 64)
	if err != nil {
		log.Fatal("error parsing int from environment", "err", err)
	}
	return result
}

func Bas64DecodeOrDie(s string) []byte {
	bytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		log.Fatal("invalid base64 string", "input", s)
	}

	return bytes
}
