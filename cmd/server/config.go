package main

import (
	"fmt"
	"strconv"
)

const defaultPort = "8080"

type config struct {
	Addr string
}

func loadConfig(getenv func(string) string) (config, error) {
	port := getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return config{}, fmt.Errorf("PORT must be a number between 1 and 65535, got %q", port)
	}
	return config{Addr: ":" + port}, nil
}
