package config

import (
	"context"
	"os"

	"github.com/HannahMarsh/PrettyLogger"
	"github.com/ilyakaznacheev/cleanenv"
)

type BulletinBoard struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Node struct {
	ID      int    `yaml:"id"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	IsMixer bool   `yaml:"is_mixer"`
}

type Client struct {
	ID   int    `yaml:"id"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Metrics struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Config struct {
	ServerLoad        int           `yaml:"server_load"`
	HeartbeatInterval int           `yaml:"heartbeat_interval"`
	MinNodes          int           `yaml:"min_nodes"`
	Epsilon           float64       `yaml:"epsilon"`
	Delta             float64       `yaml:"delta"`
	Rounds            int           `yaml:"num_rounds"`
	MinTotalMessages  int           `yaml:"min_total_messages"`
	BulletinBoard     BulletinBoard `yaml:"bulletin_board"`
	Nodes             []Node        `yaml:"nodes"`
	Metrics           Metrics       `yaml:"metrics"`
	Clients           []Client      `yaml:"clients"`
}

var GlobalConfig *Config
var GlobalCtx context.Context
var GlobalCancel context.CancelFunc

func InitGlobal() error {
	GlobalCtx, GlobalCancel = context.WithCancel(context.Background())

	GlobalConfig = &Config{}

	if dir, err := os.Getwd(); err != nil {
		return PrettyLogger.WrapError(err, "config.NewConfig(): global config error")
	} else if err2 := cleanenv.ReadConfig(dir+"/config/config.yml", GlobalConfig); err2 != nil {
		return PrettyLogger.WrapError(err2, "config.NewConfig(): global config error")
	} else if err3 := cleanenv.ReadEnv(GlobalConfig); err3 != nil {
		return PrettyLogger.WrapError(err3, "config.NewConfig(): global config error")
	} else {
		return nil
	}
}