package entity

import (
	"testing"
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
		{"Enabled", PluginStateEnabled, true},
		{"Installed", PluginStateInstalled, false},
		{"Disabled", PluginStateDisabled, false},
		{"Uninstalled", PluginStateUninstalled, false},
		{"Error", PluginStateError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plugin{State: tt.state}
			if p.IsEnabled() != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v for state %v", p.IsEnabled(), tt.expected, tt.state)
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
		name     string
		state    PluginState
		expected string
	}{
		{"Installed", PluginStateInstalled, "INSTALLED"},
		{"Enabled", PluginStateEnabled, "ENABLED"},
		{"Disabled", PluginStateDisabled, "DISABLED"},
		{"Uninstalled", PluginStateUninstalled, "UNINSTALLED"},
		{"Error", PluginStateError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("PluginState = %v, want %v", tt.state, tt.expected)
			}
		})
	}
}

func TestPluginType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		pType    PluginType
		expected string
	}{
		{"RESTEndpoint", PluginTypeRestEndpoint, "REST_ENDPOINT"},
		{"Service", PluginTypeService, "SERVICE"},
		{"EventListener", PluginTypeEventListener, "EVENT_LISTENER"},
		{"ScheduledJob", PluginTypeScheduledJob, "SCHEDULED_JOB"},
		{"SSRView", PluginTypeSSRView, "SSR_VIEW"},
		{"Middleware", PluginTypeMiddleware, "MIDDLEWARE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.pType) != tt.expected {
				t.Errorf("PluginType = %v, want %v", tt.pType, tt.expected)
			}
		})
	}
}

func TestPlugin_Fields(t *testing.T) {
	p := Plugin{
		ID:          1,
		Key:         "my-plugin",
		Name:        "My Plugin",
		Description: "A test plugin",
		Version:     "1.0.0",
		Author:      "Test Author",
		Type:        PluginTypeService,
		State:       PluginStateInstalled,
	}

	if p.Key != "my-plugin" {
		t.Errorf("Key = %v, want my-plugin", p.Key)
	}
	if p.Name != "My Plugin" {
		t.Errorf("Name = %v, want My Plugin", p.Name)
	}
	if p.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0", p.Version)
	}
	if p.Type != PluginTypeService {
		t.Errorf("Type = %v, want SERVICE", p.Type)
	}
}
