package config

import "fmt"

type Config struct {
	NetworkTimeoutSec int
	StorageWarn       int
	StorageCrit       int
	MemWarn           int
	MemCrit           int
}

func Default() Config {
	return Config{NetworkTimeoutSec: 5, StorageWarn: 80, StorageCrit: 90, MemWarn: 80, MemCrit: 95}
}

func Load(path string) (Config, error) {
	if path == "" {
		return Default(), nil
	}
	return Default(), nil
}

func (c Config) Validate() error {
	if c.NetworkTimeoutSec <= 0 {
		return fmt.Errorf("NetworkTimeoutSec must be > 0, got %d", c.NetworkTimeoutSec)
	}
	if c.StorageWarn <= 0 || c.StorageWarn >= c.StorageCrit || c.StorageCrit > 100 {
		return fmt.Errorf("storage thresholds must satisfy 0 < Warn < Crit <= 100, got Warn=%d Crit=%d", c.StorageWarn, c.StorageCrit)
	}
	if c.MemWarn <= 0 || c.MemWarn >= c.MemCrit || c.MemCrit > 100 {
		return fmt.Errorf("mem thresholds must satisfy 0 < Warn < Crit <= 100, got Warn=%d Crit=%d", c.MemWarn, c.MemCrit)
	}
	return nil
}
