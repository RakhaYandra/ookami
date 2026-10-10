package config

import "testing"

func TestDefault(t *testing.T) {
	got := Default()
	want := Config{NetworkTimeoutSec: 5, StorageWarn: 80, StorageCrit: 90, MemWarn: 80, MemCrit: 95}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadEmpty(t *testing.T) {
	got, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := Default(); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestValidate(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("Default invalid: %v", err)
	}
	bad := Default()
	bad.StorageWarn = bad.StorageCrit
	if err := bad.Validate(); err == nil {
		t.Fatal("Warn>=Crit should be invalid")
	}
}
