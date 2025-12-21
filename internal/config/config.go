package config

// Config defines game tuning values.
type Config struct {
	BaseScrapProduction          uint64
	CraftComponentTechnologyCost uint64
	CraftComponentCost           uint64
	CraftDurationSecs            uint64
}

// Default returns the standard game configuration.
func Default() Config {
	return Config{
		BaseScrapProduction:          1,
		CraftComponentTechnologyCost: 10,
		CraftComponentCost:           10,
		CraftDurationSecs:            10,
	}
}
