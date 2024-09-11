package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/HannahMarsh/PrettyLogger"
	"github.com/ilyakaznacheev/cleanenv"
)

type BulletinBoard struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Address string
}

type Relay struct {
	ID             int    `yaml:"id"`
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	PrometheusPort int    `yaml:"prometheus_port"`
	Address        string
}

type Client struct {
	ID             int    `yaml:"id"`
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	PrometheusPort int    `yaml:"prometheus_port"`
	Address        string
}

type Metrics struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Address string
}

type Config struct {
	ServerLoad              int           `yaml:"x"`
	D                       int           `yaml:"d"`
	Delta                   float64       `yaml:"delta"`
	L1                      int           `yaml:"l1"`
	L2                      int           `yaml:"l2"`
	Chi                     float64       `yaml:"chi"`
	BulletinBoard           BulletinBoard `yaml:"bulletin_board"`
	Relays                  []Relay       `yaml:"relays"`
	Metrics                 Metrics       `yaml:"visualizer"`
	Clients                 []Client      `yaml:"clients"`
	Vis                     bool          `yaml:"vis"`
	ScrapeInterval          int           `yaml:"scrapeInterval"`
	DropAllOnionsFromClient int           `yaml:"dropAllOnionsFromClient"`
	PrometheusPath          string        `yaml:"prometheusPath"`
}

var GlobalConfig *Config
var GlobalCtx context.Context
var GlobalCancel context.CancelFunc
var Names sync.Map

func InitGlobal() (error, string) {
	GlobalCtx, GlobalCancel = context.WithCancel(context.Background())

	GlobalConfig = &Config{}

	path := ""

	if dir, err := os.Getwd(); err != nil {
		return PrettyLogger.WrapError(err, "config.NewConfig(): global config error"), ""
	} else if err2 := cleanenv.ReadConfig(dir+"/config/config.yml", GlobalConfig); err2 != nil {

		// Get the absolute path of the current file
		_, currentFile, _, ok := runtime.Caller(0)
		if !ok {
			return PrettyLogger.NewError("Failed to get current file path"), ""
		}
		currentDir := filepath.Dir(currentFile)
		configFilePath := filepath.Join(currentDir, "/config.yml")
		if err3 := cleanenv.ReadConfig(configFilePath, GlobalConfig); err3 != nil {
			return PrettyLogger.WrapError(err3, "config.NewConfig(): global config error"), ""
		} else {
			path = configFilePath
		}
	} else {
		path = dir + "/config/config.yml"
		if err3 := cleanenv.ReadEnv(GlobalConfig); err3 != nil {
			return PrettyLogger.WrapError(err3, "config.NewConfig(): global config error"), ""
		}
	}

	path = strings.ReplaceAll(path, "config.yml", "prometheus.yml")
	// Update relay addresses
	for i := range GlobalConfig.Relays {
		GlobalConfig.Relays[i].Address = fmt.Sprintf("http://%s:%d", GlobalConfig.Relays[i].Host, GlobalConfig.Relays[i].Port)
	}

	// Update client addresses
	for i := range GlobalConfig.Clients {
		GlobalConfig.Clients[i].Address = fmt.Sprintf("http://%s:%d", GlobalConfig.Clients[i].Host, GlobalConfig.Clients[i].Port)
	}

	GlobalConfig.BulletinBoard.Address = fmt.Sprintf("http://%s:%d", GlobalConfig.BulletinBoard.Host, GlobalConfig.BulletinBoard.Port)
	GlobalConfig.Metrics.Address = fmt.Sprintf("http://%s:%d", GlobalConfig.Metrics.Host, GlobalConfig.Metrics.Port)
	return nil, path
}

func HostPortToName(host string, port int) string {
	return AddressToName(fmt.Sprintf("http://%s:%d", host, port))
}

var PurpleColor = "\033[35m"
var OrangeColor = "\033[33m"
var ResetColor = "\033[0m"

func AddressToName(address string) string {
	if name, ok := Names.Load(address); ok {
		return name.(string)
	}
	if strings.Count(address, "/") > 2 {
		spl := strings.Split(address, "/")
		address = spl[0] + "//" + spl[1]
	}
	if name, ok := Names.Load(address); ok {
		return name.(string)
	}
	for _, relay := range GlobalConfig.Relays {
		if address == relay.Address {
			name := fmt.Sprintf("%sRelay %d%s", PurpleColor, relay.ID, ResetColor)
			//name := fmt.Sprintf("Relay %d", relay.ID)
			Names.Store(address, name)
			return name
		}
	}
	for _, client := range GlobalConfig.Clients {
		if address == client.Address {
			name := fmt.Sprintf("%sClient %d%s", OrangeColor, client.ID, ResetColor)
			//name := fmt.Sprintf("Client %d", client.ID)
			Names.Store(address, name)
			return name
		}
	}
	if address == GlobalConfig.BulletinBoard.Address {
		name := "Bulletin Board"
		Names.Store(address, name)
		return name
	}
	if address == GlobalConfig.Metrics.Address {
		name := "Metrics"
		Names.Store(address, name)
		return name
	}
	return address
}
