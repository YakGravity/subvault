package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGrossCost(t *testing.T) {
	tests := []struct {
		name      string
		cost      float64
		taxRate   float64
		priceType string
		expected  float64
	}{
		{
			name:      "gross input - returns same cost",
			cost:      119.00,
			taxRate:   19,
			priceType: "gross",
			expected:  119.00,
		},
		{
			name:      "net input 19% - calculates gross",
			cost:      100.00,
			taxRate:   19,
			priceType: "net",
			expected:  119.00,
		},
		{
			name:      "net input 7% - calculates gross",
			cost:      100.00,
			taxRate:   7,
			priceType: "net",
			expected:  107.00,
		},
		{
			name:      "zero tax rate - returns same cost",
			cost:      100.00,
			taxRate:   0,
			priceType: "gross",
			expected:  100.00,
		},
		{
			name:      "net input zero tax - returns same cost",
			cost:      100.00,
			taxRate:   0,
			priceType: "net",
			expected:  100.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Subscription{
				Cost:      tt.cost,
				TaxRate:   tt.taxRate,
				PriceType: tt.priceType,
			}
			assert.InDelta(t, tt.expected, s.GrossCost(), 0.01)
		})
	}
}

func TestNetCost(t *testing.T) {
	tests := []struct {
		name      string
		cost      float64
		taxRate   float64
		priceType string
		expected  float64
	}{
		{
			name:      "net input - returns same cost",
			cost:      100.00,
			taxRate:   19,
			priceType: "net",
			expected:  100.00,
		},
		{
			name:      "gross input 19% - calculates net",
			cost:      119.00,
			taxRate:   19,
			priceType: "gross",
			expected:  100.00,
		},
		{
			name:      "gross input 7% - calculates net",
			cost:      107.00,
			taxRate:   7,
			priceType: "gross",
			expected:  100.00,
		},
		{
			name:      "zero tax rate - returns same cost",
			cost:      100.00,
			taxRate:   0,
			priceType: "gross",
			expected:  100.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Subscription{
				Cost:      tt.cost,
				TaxRate:   tt.taxRate,
				PriceType: tt.priceType,
			}
			assert.InDelta(t, tt.expected, s.NetCost(), 0.01)
		})
	}
}

func TestTaxAmount(t *testing.T) {
	tests := []struct {
		name      string
		cost      float64
		taxRate   float64
		priceType string
		expected  float64
	}{
		{
			name:      "gross 19% - tax is 19/119 of cost",
			cost:      119.00,
			taxRate:   19,
			priceType: "gross",
			expected:  19.00,
		},
		{
			name:      "net 19% - tax is 19% of net",
			cost:      100.00,
			taxRate:   19,
			priceType: "net",
			expected:  19.00,
		},
		{
			name:      "zero tax rate",
			cost:      100.00,
			taxRate:   0,
			priceType: "gross",
			expected:  0.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Subscription{
				Cost:      tt.cost,
				TaxRate:   tt.taxRate,
				PriceType: tt.priceType,
			}
			assert.InDelta(t, tt.expected, s.TaxAmount(), 0.01)
		})
	}
}
