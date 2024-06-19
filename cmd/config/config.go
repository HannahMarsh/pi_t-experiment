package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type BulletinBoard struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Node struct {
	ID   int    `yaml:"id"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type GlobalConfig struct {
	ServerLoad        int           `yaml:"server_load"`
	HeartbeatInterval int           `yaml:"heartbeat_interval"`
	MinNodes          int           `yaml:"min_nodes"`
	Epsilon           float64       `yaml:"epsilon"`
	Delta             float64       `yaml:"delta"`
	Rounds            int           `yaml:"rounds"`
	MinQueueLength    int           `yaml:"min_queue_length"`
	BulletinBoard     BulletinBoard `yaml:"bulletin_board"`
	Nodes             []Node        `yaml:"nodes"`
}

func NewConfig() (*GlobalConfig, error) {
	cfg := &GlobalConfig{}

	if dir, err := os.Getwd(); err != nil {
		return nil, fmt.Errorf("config.NewConfig(): global config error: %w", err)
	} else if err2 := cleanenv.ReadConfig(dir+"/cmd/config/config.yml", cfg); err2 != nil {
		return nil, fmt.Errorf("config.NewConfig(): global config error: %w", err2)
	} else if err3 := cleanenv.ReadEnv(cfg); err3 != nil {
		return nil, fmt.Errorf("config.NewConfig(): global config error: %w", err3)
	} else {
		return cfg, nil
	}
}
