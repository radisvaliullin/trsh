package rstat

import (
	"fmt"
	"os"
)

const (
	RSTAT_TOKEN     = "RSTAT_TOKEN"
	RSTAT_SUBREDDIT = "RSTAT_SUBREDDIT"
)

type Config struct {
	Client ClientConfig
}

func GetConfig() (Config, error) {
	conf := Config{}

	// Client Config
	token := os.Getenv(RSTAT_TOKEN)
	if len(token) == 0 {
		return conf, fmt.Errorf("config: missed set %v env variable", RSTAT_TOKEN)
	}
	subreddit := os.Getenv(RSTAT_SUBREDDIT)
	if len(subreddit) == 0 {
		return conf, fmt.Errorf("config: missed set %v env variable", RSTAT_SUBREDDIT)
	}
	conf.Client = ClientConfig{
		Token:     token,
		Subreddit: subreddit,
	}

	return conf, nil
}
