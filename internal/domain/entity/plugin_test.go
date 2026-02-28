package entity

import (
	"testing"
	"time"
)

func TestPlugin_TableName(t *testing.T) {
	p := Plugin{}
	if p.TableName() != "plugins" {
		t.Errorf("TableName() = %v, want plugins", p.TableName())
	}
}

func TestPlugin_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		state    PluginState
		expected bool
	}{
		{"enabled state", PluginStateEnabled, true},
		{"installed state", PluginStateInstalled, false},
		{"disabled state", PluginStateDisabled, false},
		{"uninstalled state", PluginStateUninstalled, false},
		{"error state", PluginStateError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plugin{State: tt.state}
			if got := p.IsEnabled(); got != tt.expected {
				t.Errorf("Plugin.IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPluginExtension_TableName(t *testing.T) {
	pe := PluginExtension{}
	if pe.TableName() != "plugin_extensions" {
		t.Errorf("TableName() = %v, want plugin_extensions", pe.TableName())
	}
}

func TestPluginState_Constants(t *testing.T) {
	tests := []struct {
		state    PluginState
		expected string
	}{
		{PluginStateInstalled, "INSTALLED"},
		{PluginStateEnabled, "ENABLED"},
		{PluginStateDisabled, "DISABLED"},
		{PluginStateUninstalled, "UNINSTALLED"},
		{PluginStateError, "ERROR"},
	}
	for _, tt := range tests {
		if string(tt.state) != tt.expected {
			t.Errorf("PluginState = %v, want %v", tt.state, tt.expected)
		}
	}
}

func TestPluginType_Constants(t *testing.T) {
	tests := []struct {
		pluginType PluginType
		expected   string
	}{
		{PluginTypeRestEndpoint, "REST_ENDPOINT"},
		{PluginTypeService, "SERVICE"},
		{PluginTypeEventListener, "EVENT_LISTENER"},
		{PluginTypeScheduledJob, "SCHEDULED_JOB"},
		{PluginTypeSSRView, "SSR_VIEW"},
		{PluginTypeMiddleware, "MIDDLEWARE"},
	}
	for _, tt := range tests {
		if string(tt.pluginType) != tt.expected {
			t.Errorf("PluginType = %v, want %v", tt.pluginType, tt.expected)
		}
	}
}

func TestPlugin_Fields(t *testing.T) {
	now := time.Now()
	p := &Plugin{
		ID:          1,
		Key:         "test-plugin",
		Name:        "Test Plugin",
		Description: "A test plugin",
		Version:     "1.0.0",
		Author:      "Test Author",
		Type:        PluginTypeService,
		State:       PluginStateInstalled,
		Config:      `{"key":"value"}`,
		Checksum:    "abc123",
		Path:        "/plugins/test",
		InstalledAt: now,
		EnabledAt:   &now,
	}

	if p.Key != "test-plugin" {
		t.Errorf("Key = %v, want test-plugin", p.Key)
	}
	if p.State != PluginStateInstalled {
		t.Errorf("State = %v, want INSTALLED", p.State)
	}
	if p.Type != PluginTypeService {
		t.Errorf("Type = %v, want SERVICE", p.Type)
	}
	if p.IsEnabled() {
		t.Error("IsEnabled() should be false for INSTALLED state")
	}
}

func TestPluginExtension_Fields(t *testing.T) {
	pe := &PluginExtension{
		ID:       1,
		PluginID: 1,
		Name:     "Test Extension",
		Type:     PluginTypeRestEndpoint,
		Path:     "/api/test",
		Handler:  "handleTest",
		Config:   `{}`,
	}

	if pe.Name != "Test Extension" {
		t.Errorf("Name = %v, want Test Extension", pe.Name)
	}
	if pe.Type != PluginTypeRestEndpoint {
		t.Errorf("Type = %v, want REST_ENDPOINT", pe.Type)
	}
}
