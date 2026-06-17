package model

import "testing"

// TestApplyEnvOverridesNoopWhenUnset verifies that with no environment variables
// set, ApplyEnvOverrides leaves the config untouched (preserving file-only
// behaviour).
func TestApplyEnvOverridesNoopWhenUnset(t *testing.T) {
	c := Config{}
	c.Server.WebSocket.Host = "127.0.0.1"
	c.Server.WebSocket.Port = 8080
	c.Server.Authentication.Enable = false

	c.ApplyEnvOverrides()

	if c.Server.WebSocket.Host != "127.0.0.1" || c.Server.WebSocket.Port != 8080 || c.Server.Authentication.Enable {
		t.Fatalf("env-unset override changed config: %+v", c.Server)
	}
}

// TestApplyEnvOverridesApplies verifies that set environment variables override
// the corresponding runtime fields.
func TestApplyEnvOverridesApplies(t *testing.T) {
	t.Setenv("AIWOLF_HOST", "0.0.0.0")
	t.Setenv("AIWOLF_PORT", "9999")
	t.Setenv("AIWOLF_AUTH_ENABLE", "true")
	t.Setenv("AIWOLF_JSON_LOG_DIR", "/data/json")

	c := Config{}
	c.ApplyEnvOverrides()

	if c.Server.WebSocket.Host != "0.0.0.0" {
		t.Errorf("host: got %q", c.Server.WebSocket.Host)
	}
	if c.Server.WebSocket.Port != 9999 {
		t.Errorf("port: got %d", c.Server.WebSocket.Port)
	}
	if !c.Server.Authentication.Enable {
		t.Errorf("auth: expected true")
	}
	if c.JSONLogger.OutputDir != "/data/json" {
		t.Errorf("json dir: got %q", c.JSONLogger.OutputDir)
	}
}
