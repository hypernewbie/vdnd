package llm

import (
	"context"
	"testing"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/state"
)

type mockManualStore struct {
	state.MemoryStore
}

func (s *mockManualStore) GetManual() (string, error) {
	return "MOCK MANUAL CONTENT", nil
}

func TestOrchestrator_GetVDManual(t *testing.T) {
	deps := cli.Deps{
		Store: &mockManualStore{},
	}
	o := NewOrchestrator(context.Background(), nil, deps)
	
	content, err := o.getVDManual()
	if err != nil {
		t.Fatalf("getVDManual failed: %v", err)
	}
	
	if content != "MOCK MANUAL CONTENT" {
		t.Errorf("Expected 'MOCK MANUAL CONTENT', got %q", content)
	}
}

func TestOrchestrator_RegisterManualTool(t *testing.T) {
	o := NewOrchestrator(context.Background(), nil, cli.Deps{})
	o.RegisterSubagents() // Register with no agents
	
	found := false
	for _, tool := range o.tools {
		if tool.Name == "get_vd_manual" {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("get_vd_manual tool not registered")
	}
}
