package config

import (
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
)

type Global struct {
	ScrapeInterval string         `yaml:"scrape_interval"`
	ExternalLabels ExternalLabels `yaml:"external_labels"`
}

type ExternalLabels struct {
	Monitor string `yaml:"monitor"`
}

type ScrapeConfig struct {
	JobName        string         `yaml:"job_name"`
	ScrapeInterval string         `yaml:"scrape_interval"`
	StaticConfigs  []StaticConfig `yaml:"static_configs"`
}

type StaticConfig struct {
	Targets []string `yaml:"targets"`
}

type PromConfig struct {
	Global        Global         `yaml:"global"`
	ScrapeConfigs []ScrapeConfig `yaml:"scrape_configs"`
}

func InitPrometheusConfig(path string) error {

	promCfg := PromConfig{
		Global: Global{
			ScrapeInterval: "15s",
			ExternalLabels: ExternalLabels{
				Monitor: "pi_t",
			},
		},
		ScrapeConfigs: []ScrapeConfig{},
	}

	for _, client := range GlobalConfig.Clients {
		promCfg.ScrapeConfigs = append(promCfg.ScrapeConfigs, ScrapeConfig{
			JobName:        fmt.Sprintf("client-%d", client.ID),
			ScrapeInterval: "5s",
			StaticConfigs: []StaticConfig{
				{
					Targets: []string{fmt.Sprintf("%s:%d", client.Host, client.PrometheusPort)},
				},
			},
		})
	}

	for _, relay := range GlobalConfig.Relays {
		promCfg.ScrapeConfigs = append(promCfg.ScrapeConfigs, ScrapeConfig{
			JobName:        fmt.Sprintf("relay-%d", relay.ID),
			ScrapeInterval: "5s",
			StaticConfigs: []StaticConfig{
				{
					Targets: []string{fmt.Sprintf("%s:%d", relay.Host, relay.PrometheusPort)},
				},
			},
		})
	}

	// Marshal the struct into YAML format
	data, err := yaml.Marshal(&promCfg)
	if err != nil {
		return pl.WrapError(err, "failed to marshal prometheus config")
	}

	// Open the file for writing (creates the file if it doesn't exist)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return pl.WrapError(err, "failed to open file for writing")
	}
	defer file.Close()

	// Write the YAML data to the file
	_, err = file.Write(data)
	if err != nil {
		return pl.WrapError(err, "failed to write prometheus config to file")
	}

	// Ensure the data is flushed to disk immediately
	err = file.Sync()
	if err != nil {
		return pl.WrapError(err, "failed to flush prometheus config to disk")
	}

	slog.Info("prometheus config written to file", "path", path)

	return nil
}
