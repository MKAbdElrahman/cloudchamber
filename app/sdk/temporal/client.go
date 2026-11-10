// Package temporal provides Temporal client configuration and dialing helpers.
package temporal

import (
	"fmt"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
)

// Config holds Temporal connection settings.
type Config struct {
	HostPort     string
	Namespace    string
	Interceptors []interceptor.ClientInterceptor
}

// NewClient dials a Temporal client with the provided configuration.
func NewClient(cfg Config) (client.Client, error) {
	if cfg.HostPort == "" {
		cfg.HostPort = client.DefaultHostPort
	}
	if cfg.Namespace == "" {
		cfg.Namespace = client.DefaultNamespace
	}

	c, err := client.Dial(client.Options{
		HostPort:     cfg.HostPort,
		Namespace:    cfg.Namespace,
		Interceptors: cfg.Interceptors,
	})
	if err != nil {
		return nil, fmt.Errorf("dial temporal host=%s namespace=%s: %w", cfg.HostPort, cfg.Namespace, err)
	}

	return c, nil
}
