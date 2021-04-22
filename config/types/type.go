package types

import (
        "flag"
        "jaeger-s3/config"
        "github.com/cortexproject/cortex/pkg/util/flagext"
)

type Config struct {
        config.Config     `yaml:",inline"`
        printVersion    bool
        verifyConfig    bool
        printConfig     bool
        logConfig       bool
        configFile      string
        configExpandEnv bool
}

func (c *Config) RegisterFlags(f *flag.FlagSet) {
        f.BoolVar(&c.printVersion, "version", false, "Print this builds version information")
        f.BoolVar(&c.verifyConfig, "verify-config", false, "Verify config file and exits")
        f.BoolVar(&c.printConfig, "print-config-stderr", false, "Dump the entire Loki config object to stderr")
        f.BoolVar(&c.logConfig, "log-config-reverse-order", false, "Dump the entire Loki config object at Info log "+
                "level with the order reversed, reversing the order makes viewing the entries easier in Grafana.")
        f.StringVar(&c.configFile, "config", "", "yaml file to load")
        f.BoolVar(&c.configExpandEnv, "config.expand-env", false, "Expands ${var} in config according to the values of the environment variables.")
        c.Config.RegisterFlags(f)
}

func (c *Config) Clone() flagext.Registerer {
        return func(c Config) *Config {
                return &c
        }(*c)
}

