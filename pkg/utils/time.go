package utils

import (
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func GetTimestamp() time.Time {
	// Get the current time
	now := time.Now()

	// Convert to UTC
	utcTime := now.UTC()

	return utcTime
}

func installNTP() (string, error) {
	service := "ntp"
	// Command to install NTP using apt
	installCmd := exec.Command("sudo", "apt", "install", "-y", "ntp")

	// Run the command to install NTP
	output, err := installCmd.CombinedOutput() // CombinedOutput captures both stdout and stderr

	if err != nil {
		service = "chronyd"
		installCmd = exec.Command("sudo", "apt", "install", "-y", service)

		// Run the command to install NTP
		output, err = installCmd.CombinedOutput() // CombinedOutput captures both stdout and stderr

		if err != nil {
			return "", pl.WrapError(err, "Failed to install NTP or chronyd. Output: %s", string(output))
		}
	}

	// Print the output from the installation command
	fmt.Printf("Command output: %s\n", output)

	slog.Info(service + " installation completed successfully.")
	return service, nil
}

// StartNTP starts the NTP service if it is not already running
// and returns a function to stop the NTP service when called.
func StartNTP() (stopNTP func()) {
	// Check if the operating system is Linux
	if runtime.GOOS != "linux" {
		slog.Info(runtime.GOOS + " system detected. With non-linux OS, no need to run NTP client; synchronization is handled automatically.")
		return func() {
			slog.Info("No NTP service to stop since the program isn't running on Linux.")
		}
	}

	service := "ntp"

	// Command to check if ntpd is installed (or replace with chronyd if using chrony)
	checkInstallCmd := exec.Command("which", "ntpd")

	if err := checkInstallCmd.Run(); err != nil {
		// If ntpd is not found, check for chronyd
		checkInstallCmd = exec.Command("which", "chronyd")
		if err := checkInstallCmd.Run(); err != nil {
			slog.Info("Neither NTP client nor Chrony is installed. Installing NTP...")
			service, err = installNTP()
			if err != nil {
				slog.Error("Failed to install NTP client", err)
				return func() {}
			}
		} else {
			service = "chronyd"
		}
	}

	// Command to check if the NTP service is running
	checkCmd := exec.Command("sudo", "service", service, "status")
	output, err := checkCmd.Output()

	if err != nil {
		slog.Error("Failed to check "+service+" service status", err)
		return func() {}
	}

	// Check if the output contains "active (running)" (for systems using service)
	if strings.Contains(string(output), "active (running)") {
		slog.Info("NTP service is already running.")
		// Return a no-op stop function since NTP was already running
		return func() {
			slog.Info("NTP service was started by a different program. No need to stop it.")
		}
	}

	// If NTP is not running, attempt to start it
	slog.Info(service + " service is not running, attempting to start it...")
	startCmd := exec.Command("sudo", "service", service, "start")

	// Run the command to start the service
	if err := startCmd.Start(); err != nil {
		slog.Error("Failed to start "+service+" service", err)
		return func() {}
	}

	// Wait for the NTP service to start
	if err := startCmd.Wait(); err != nil {
		slog.Error("Error while waiting for "+service+" service to start", err)
		return func() {}
	}

	slog.Info("NTP service started successfully.")

	// Return a function to stop the NTP service when done
	return func() {
		slog.Info("Stopping NTP service...")
		stopCmd := exec.Command("sudo", "service", service, "stop")

		// Run the command to stop the service
		if err := stopCmd.Start(); err != nil {
			slog.Error("Failed to stop "+service+" service", err)
			return
		}

		// Wait for the command to complete
		if err := stopCmd.Wait(); err != nil {
			slog.Error("Error while waiting for "+service+" service to stop", err)
			return
		}

		slog.Info(service + " service stopped successfully.")
	}
}
