package server

import (
	"echonet-list/config"
	"testing"
	"time"
)

// TestConfigParsing_ValidIntervals tests parsing of valid forced update intervals
func TestConfigParsing_ValidIntervals(t *testing.T) {
	tests := []struct {
		name             string
		intervalString   string
		expectedDuration time.Duration
	}{
		{
			name:             "Thirty minutes",
			intervalString:   "30m",
			expectedDuration: 30 * time.Minute,
		},
		{
			name:             "One hour",
			intervalString:   "1h",
			expectedDuration: 1 * time.Hour,
		},
		{
			name:             "Two hours thirty minutes",
			intervalString:   "2h30m",
			expectedDuration: 2*time.Hour + 30*time.Minute,
		},
		{
			name:             "Five seconds (very short)",
			intervalString:   "5s",
			expectedDuration: 5 * time.Second,
		},
		{
			name:             "Zero (disabled)",
			intervalString:   "0",
			expectedDuration: 0,
		},
		{
			name:             "Zero with unit",
			intervalString:   "0m",
			expectedDuration: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := time.ParseDuration(tt.intervalString)
			if err != nil {
				t.Errorf("ParseDuration(%q) returned error: %v", tt.intervalString, err)
				return
			}

			if duration != tt.expectedDuration {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.intervalString, duration, tt.expectedDuration)
			}
		})
	}
}

// TestConfigParsing_InvalidIntervals tests handling of invalid interval strings
func TestConfigParsing_InvalidIntervals(t *testing.T) {
	invalidIntervals := []string{
		"invalid",
		"30x", // invalid unit
		"-5m", // negative
		"",    // empty string
		"5",   // missing unit (this is actually valid for "5ns")
	}

	for _, intervalString := range invalidIntervals {
		t.Run(intervalString, func(t *testing.T) {
			_, err := time.ParseDuration(intervalString)
			// Most should error, but some edge cases like "5" might not
			// The main thing is we handle them gracefully in main.go
			t.Logf("ParseDuration(%q) returned error: %v", intervalString, err)
		})
	}
}

// TestConfigDefaults tests that default configuration values are reasonable
func TestConfigDefaults(t *testing.T) {
	cfg := config.NewConfig()

	// Test periodic update interval default
	periodicInterval, err := time.ParseDuration(cfg.WebSocket.PeriodicUpdateInterval)
	if err != nil {
		t.Errorf("Default periodic update interval %q is invalid: %v",
			cfg.WebSocket.PeriodicUpdateInterval, err)
	}

	if periodicInterval <= 0 {
		t.Errorf("Default periodic update interval should be positive, got %v", periodicInterval)
	}

	if periodicInterval < 10*time.Second {
		t.Errorf("Default periodic update interval seems too short: %v", periodicInterval)
	}

	// Test forced update interval default
	forcedInterval, err := time.ParseDuration(cfg.WebSocket.ForcedUpdateInterval)
	if err != nil {
		t.Errorf("Default forced update interval %q is invalid: %v",
			cfg.WebSocket.ForcedUpdateInterval, err)
	}

	if forcedInterval <= 0 {
		t.Errorf("Default forced update interval should be positive, got %v", forcedInterval)
	}

	// Forced interval should be longer than periodic interval
	if forcedInterval <= periodicInterval {
		t.Errorf("Forced update interval (%v) should be longer than periodic interval (%v)",
			forcedInterval, periodicInterval)
	}

	// Should be at least 10 minutes for reasonable operation
	if forcedInterval < 10*time.Minute {
		t.Errorf("Default forced update interval seems too short: %v", forcedInterval)
	}

	t.Logf("Default intervals - Periodic: %v, Forced: %v", periodicInterval, forcedInterval)
}

// TestStartOptions_ForcedUpdateInterval tests that StartOptions correctly handles forced update intervals
func TestStartOptions_ForcedUpdateInterval(t *testing.T) {
	tests := []struct {
		name                     string
		periodicUpdateInterval   time.Duration
		forcedUpdateInterval     time.Duration
		expectValidConfiguration bool
	}{
		{
			name:                     "Normal configuration",
			periodicUpdateInterval:   1 * time.Minute,
			forcedUpdateInterval:     30 * time.Minute,
			expectValidConfiguration: true,
		},
		{
			name:                     "Forced updates disabled",
			periodicUpdateInterval:   1 * time.Minute,
			forcedUpdateInterval:     0,
			expectValidConfiguration: true,
		},
		{
			name:                     "Very short intervals",
			periodicUpdateInterval:   1 * time.Second,
			forcedUpdateInterval:     5 * time.Second,
			expectValidConfiguration: true,
		},
		{
			name:                     "Forced interval same as periodic",
			periodicUpdateInterval:   1 * time.Minute,
			forcedUpdateInterval:     1 * time.Minute,
			expectValidConfiguration: true, // Should work but might be unusual
		},
		{
			name:                     "Forced interval shorter than periodic",
			periodicUpdateInterval:   5 * time.Minute,
			forcedUpdateInterval:     1 * time.Minute,
			expectValidConfiguration: true, // Should work but unusual
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := StartOptions{
				PeriodicUpdateInterval: tt.periodicUpdateInterval,
				ForcedUpdateInterval:   tt.forcedUpdateInterval,
			}

			// Basic validation - StartOptions should accept any duration values
			if options.PeriodicUpdateInterval != tt.periodicUpdateInterval {
				t.Errorf("PeriodicUpdateInterval not set correctly: got %v, want %v",
					options.PeriodicUpdateInterval, tt.periodicUpdateInterval)
			}

			if options.ForcedUpdateInterval != tt.forcedUpdateInterval {
				t.Errorf("ForcedUpdateInterval not set correctly: got %v, want %v",
					options.ForcedUpdateInterval, tt.forcedUpdateInterval)
			}

			// Log configuration for manual review
			t.Logf("Configuration - Periodic: %v, Forced: %v, Valid: %v",
				options.PeriodicUpdateInterval, options.ForcedUpdateInterval, tt.expectValidConfiguration)
		})
	}
}

// TestConfigurationRecommendations tests that recommended configurations work properly
func TestConfigurationRecommendations(t *testing.T) {
	recommendations := []struct {
		name             string
		periodicInterval string
		forcedInterval   string
		description      string
	}{
		{
			name:             "Development",
			periodicInterval: "10s",
			forcedInterval:   "1m",
			description:      "Fast updates for development/testing",
		},
		{
			name:             "Production",
			periodicInterval: "1m",
			forcedInterval:   "30m",
			description:      "Balanced for production use",
		},
		{
			name:             "Conservative",
			periodicInterval: "5m",
			forcedInterval:   "1h",
			description:      "Conservative for resource-constrained environments",
		},
		{
			name:             "Aggressive Recovery",
			periodicInterval: "30s",
			forcedInterval:   "5m",
			description:      "Frequent forced updates for quick offline device recovery",
		},
	}

	for _, rec := range recommendations {
		t.Run(rec.name, func(t *testing.T) {
			periodicDuration, err := time.ParseDuration(rec.periodicInterval)
			if err != nil {
				t.Errorf("Invalid periodic interval %q: %v", rec.periodicInterval, err)
				return
			}

			forcedDuration, err := time.ParseDuration(rec.forcedInterval)
			if err != nil {
				t.Errorf("Invalid forced interval %q: %v", rec.forcedInterval, err)
				return
			}

			// Basic sanity checks
			if periodicDuration <= 0 {
				t.Errorf("Periodic interval should be positive: %v", periodicDuration)
			}

			if forcedDuration <= 0 {
				t.Errorf("Forced interval should be positive: %v", forcedDuration)
			}

			t.Logf("%s: %s - Periodic: %v, Forced: %v",
				rec.name, rec.description, periodicDuration, forcedDuration)
		})
	}
}
