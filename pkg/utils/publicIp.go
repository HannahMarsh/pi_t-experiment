package utils

import (
	"encoding/json"
	pl "github.com/HannahMarsh/PrettyLogger"
	"io"
	"log/slog"
	"net/http"
)

type PublicIP struct {
	IP       string `json:"ip"`
	HostName string `json:"hostname"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Location string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
	ReadMe   string `json:"readme"`
}

func GetPublicIP() (*PublicIP, error) {
	resp, err := http.Get("https://ipinfo.io/json")
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			slog.Error("failed to close response body", err)
		}
	}(resp.Body)

	var ip PublicIP
	if err := json.NewDecoder(resp.Body).Decode(&ip); err != nil {
		return nil, pl.WrapError(err, "failed to decode public IP")
	}
	return &ip, nil
}
