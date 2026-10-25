package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Logger Logger
	Mongo  struct {
		URI            string `envconfig:"MONGODB_URI" default:"mongodb://m1:27017,m2:27017,m3:27017/hw2?replicaSet=hw2rs&retryWrites=true&serverSelectionTimeoutMS=10000&readPreference=primary"`
		DBName         string `envconfig:"MONGO_DB" default:"hw2"`
		FailoverDBName string `envconfig:"MONGO_FAILOVER_DB" default:"hw2_failover"`
		ServerVersion  string `envconfig:"MONGO_SERVER_VERSION" default:"8.0.30"`
	}

	ResultsPath string `envconfig:"RESULTS_PATH" default:"results/results.json"`
}

type Logger struct {
	LogLevel  string `envconfig:"LOGGER_LEVEL" default:"debug"`
	LogFormat string `envconfig:"LOGGER_FORMAT" default:"console"`
}

func Parse() (*Config, error) {
	cfg := &Config{}

	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
