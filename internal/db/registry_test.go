package db

import "testing"

func TestRegistryNewEngine(t *testing.T) {
	_, err := NewEngine("postgres")
	if err != nil {
		t.Errorf("expected no error for postgres, got %v", err)
	}
	_, err = NewEngine("mysql")
	if err != nil {
		t.Errorf("expected no error for mysql, got %v", err)
	}
	_, err = NewEngine("sqlite")
	if err == nil {
		t.Error("expected error for unsupported engine")
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	if r.Active() != nil {
		t.Error("expected nil active engine on new registry")
	}
	if len(r.Names()) != 0 {
		t.Error("expected empty names on new registry")
	}
}
