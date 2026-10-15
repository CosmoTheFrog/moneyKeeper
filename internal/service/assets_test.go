package service

import (
	"math/big"
	"testing"
)

func TestAssetUnits(t *testing.T) {
	tests := []struct {
		name  string
		stock bool
		usd   string
		want  int
	}{
		{"bitcoin", false, "67000", 8},
		{"mid coin", false, "250", 6},
		{"cheap coin", false, "0.2", 4},
		{"very cheap coin", false, "0.0004", 2},
		{"meme coin", false, "0.000012", 0},
		{"stock keeps four", true, "0.5", 4},
		{"boundary 1000", false, "1000", 8},
		{"boundary 10", false, "10", 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usd, _ := new(big.Rat).SetString(tt.usd)
			if got := assetUnits(tt.stock, usd); got != tt.want {
				t.Errorf("assetUnits(%v, %s) = %d, want %d", tt.stock, tt.usd, got, tt.want)
			}
		})
	}
}