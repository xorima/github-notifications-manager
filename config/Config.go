package config

import (
	"os"
	"strings"
)

var AppConfig = NewConfig()

type Config struct {
	GithubToken string
	OrgName     string
	DryRun      bool
	State       string
}

func NewConfig() *Config {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		panic("GITHUB_TOKEN is not set")
	}
	return &Config{
		GithubToken: githubToken,
	}
}

func (c *Config) GetState() []string {
	return strings.Split(c.State, ",")
}
