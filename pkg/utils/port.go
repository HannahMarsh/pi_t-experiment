package utils

import (
	"log/slog"
	"net"
)

// GetAvailablePort finds an available port and returns it
func GetAvailablePort() (int, error) {
	// Create a TCP listener on port 0 to have the OS assign a free port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer func(listener net.Listener) {
		if err := listener.Close(); err != nil {
			slog.Error("", err)
		}
	}(listener)

	// Get the actual port that was assigned by the OS
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
