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

type Config struct {
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

var GlobalConfig *Config

func InitConfig() error {
	GlobalConfig = &Config{}

	if dir, err := os.Getwd(); err != nil {
		return fmt.Errorf("config.NewConfig(): global config error: %w", err)
	} else if err2 := cleanenv.ReadConfig(dir+"/cmd/config/config.yml", GlobalConfig); err2 != nil {
		return fmt.Errorf("config.NewConfig(): global config error: %w", err2)
	} else if err3 := cleanenv.ReadEnv(GlobalConfig); err3 != nil {
		return fmt.Errorf("config.NewConfig(): global config error: %w", err3)
	} else {
		return nil
	}
}
