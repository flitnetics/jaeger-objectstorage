package config

import (
	"github.com/spf13/viper"
)

const (
	dbPrefix = "db."

	flagHost     = dbPrefix + "host"
	flagUsername = dbPrefix + "username"
	flagPassword = dbPrefix + "password"
)

// Configuration describes the options to customize the storage behavior
type Configuration struct {
	// TCP host:port or Unix socket depending on Network.
	Host     string `yaml:"host"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`

	/*
		Database string `yaml:"database"`

		// Network type, either tcp or unix.
		// Default is tcp.
		Network string `yaml:"network"`

		// ApplicationName is the application name. Used in logs on Pg side.
		// Only available from pg-9.0.
		ApplicationName string `yaml:"applicationName"`

		// TLS config for secure connections.
		//TLSConfig *tls.Config `yaml:"host"`

		// Dial timeout for establishing new connections.
		// Default is 5 seconds.
		DialTimeout time.Duration `yaml:"dialTimeout"`

		// Timeout for socket reads. If reached, commands will fail
		// with a timeout instead of blocking.
		ReadTimeout time.Duration `yaml:"readTimeout"`
		// Timeout for socket writes. If reached, commands will fail
		// with a timeout instead of blocking.
		WriteTimeout time.Duration `yaml:"writeTimeout"`

		// Maximum number of retries before giving up.
		// Default is to not retry failed queries.
		MaxRetries int `yaml:"maxRetries"`
		// Whether to retry queries cancelled because of statement_timeout.
		RetryStatementTimeout bool `yaml:"retryStatementTimeout"`
		// Minimum backoff between each retry.
		// Default is 250 milliseconds; -1 disables backoff.
		MinRetryBackoff time.Duration `yaml:"minRetryBackoff"`
		// Maximum backoff between each retry.
		// Default is 4 seconds; -1 disables backoff.
		MaxRetryBackoff time.Duration `yaml:"maxRetryBackoff"`

		// Maximum number of socket connections.
		// Default is 10 connections per every CPU as reported by runtime.NumCPU.
		PoolSize int `yaml:"poolSize"`
		// Minimum number of idle connections which is useful when establishing
		// new connection is slow.
		MinIdleConns int `yaml:"minIdleConns"`
		// Connection age at which client retires (closes) the connection.
		// It is useful with proxies like PgBouncer and HAProxy.
		// Default is to not close aged connections.
		MaxConnAge time.Duration `yaml:"maxConnAge"`
		// Time for which client waits for free connection if all
		// connections are busy before returning an error.
		// Default is 30 seconds if ReadTimeOut is not defined, otherwise,
		// ReadTimeout + 1 second.
		PoolTimeout time.Duration `yaml:"poolTimeout"`
		// Amount of time after which client closes idle connections.
		// Should be less than server's timeout.
		// Default is 5 minutes. -1 disables idle timeout check.
		IdleTimeout time.Duration `yaml:"idleTimeout"`
		// Frequency of idle checks made by idle connections reaper.
		// Default is 1 minute. -1 disables idle connections reaper,
		// but idle connections are still discarded by the client
		// if IdleTimeout is set.
		IdleCheckFrequency time.Duration `yaml:"idleCheckFrequency"`
	*/
}

// InitFromViper initializes the options struct with values from Viper
func (c *Configuration) InitFromViper(v *viper.Viper) {
	c.Host = v.GetString(flagHost)
	if len(c.Host) == 0 {
		c.Host = "localhost:5432"
	}
	c.Username = v.GetString(flagUsername)
	if len(c.Username) == 0 {
		c.Username = "postgres"
	}
	c.Password = v.GetString(flagPassword)
	if len(c.Password) == 0 {
		c.Password = "changeme"
	}
}
