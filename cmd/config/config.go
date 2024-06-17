package config

import (
	"fmt"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type BulletinBoard struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Node struct {
	ID         int    `yaml:"id"`
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	PublicKey  string `yaml:"public_key"`
	PrivateKey string `yaml:"private_key"`
}

type GlobalConfig struct {
	HeartbeatInterval int           `yaml:"heartbeat_interval"`
	BulletinBoard     BulletinBoard `yaml:"bulletin_board"`
	Nodes             []Node        `yaml:"nodes"`
}

func NewConfig() (*GlobalConfig, error) {
	cfg := &GlobalConfig{}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// debug
	fmt.Println(dir)

	err = cleanenv.ReadConfig(dir+"/.../.../global_config.yml", cfg)
	if err != nil {
		return nil, fmt.Errorf("global config error: %w", err)
	}

	err = cleanenv.ReadEnv(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
