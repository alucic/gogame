package config

import "testing"

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.BaseScrapProduction != 1 {
		t.Fatalf("BaseScrapProduction: expected 1 got %d", cfg.BaseScrapProduction)
	}
	if cfg.CraftComponentTechnologyCost != 10 {
		t.Fatalf("CraftComponentTechnologyCost: expected 10 got %d", cfg.CraftComponentTechnologyCost)
	}
	if cfg.CraftComponentCost != 10 {
		t.Fatalf("CraftComponentCost: expected 10 got %d", cfg.CraftComponentCost)
	}
	if cfg.CraftDurationSecs != 10 {
		t.Fatalf("CraftDurationSecs: expected 10 got %d", cfg.CraftDurationSecs)
	}
}
