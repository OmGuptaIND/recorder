package env

import (
	"log"

	"github.com/spf13/viper"
)

type Env struct {
	Environment string `mapstructure:"ENVIRONMENT"`
}

// LoadEnvironmentVariables loads environment variables
func LoadEnvironmentVariables() (*Env, error) {
	viper.SetDefault("ENVIRONMENT", "development")

	env := &Env{}

	viper.SetConfigFile(".env")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(&env)

	if err != nil {
		log.Fatal("Environment can't be loaded: ", err)
		return nil, err
	}

	return env, nil
}

// IsDevelopment returns true if the environment is development
func IsDevelopment() bool {
	return viper.GetString("ENVIRONMENT") == "development"
}

// RecordingBucket returns the recording bucket
func GetBucketName() string {
	return viper.GetString("BUCKET_NAME")
}

func GetBucketEndpoint() string {
	return viper.GetString("BUCKET_ENDPOINT")
}

func GetBucketAppKey() string {
	return viper.GetString("BUCKET_APP_KEY")
}

func GetBucketKeyId() string {
	return viper.GetString("BUCKET_KEY_ID")
}

func GetBucketRegion() string {
	return viper.GetString("BUCKET_REGION")
}
