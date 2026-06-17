package model

import "testing"

func TestApplyEnvOverridesNoopWhenUnset(t *testing.T) {
	c := Config{}
	c.Server.WebSocket.Host = "127.0.0.1"
	c.Server.WebSocket.Port = 8080

	c.ApplyEnvOverrides()

	if c.Server.WebSocket.Host != "127.0.0.1" || c.Server.WebSocket.Port != 8080 {
		t.Fatalf("env-unset override changed config: %+v", c.Server)
	}
}

func TestApplyEnvOverridesApplies(t *testing.T) {
	t.Setenv("HOST", "0.0.0.0")
	t.Setenv("PORT", "9999")

	c := Config{}
	c.ApplyEnvOverrides()

	if c.Server.WebSocket.Host != "0.0.0.0" {
		t.Errorf("host: got %q", c.Server.WebSocket.Host)
	}
	if c.Server.WebSocket.Port != 9999 {
		t.Errorf("port: got %d", c.Server.WebSocket.Port)
	}
}
