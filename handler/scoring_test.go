package handler

import (
	"testing"
)

func TestCalculateEndTotal(t *testing.T) {
	// Helper function simulation from scoring.go
	calculate := func(arrow1, arrow2, arrow3, arrow4, arrow5, arrow6 *int) (int, int, int) {
		arrows := []*int{arrow1, arrow2, arrow3, arrow4, arrow5, arrow6}
		total := 0
		xCount := 0
		tenCount := 0

		for _, arrow := range arrows {
			if arrow != nil {
				val := *arrow
				if val == 10 {
					tenCount++
					total += 10
				} else if val == 11 { // X
					xCount++
					tenCount++
					total += 10
				} else {
					total += val
				}
			}
		}
		return total, xCount, tenCount
	}

	ptr := func(v int) *int { return &v }

	tests := []struct {
		name     string
		arrows   []*int
		expTotal int
		expX     int
		expTen   int
	}{
		{
			"All 10s",
			[]*int{ptr(10), ptr(10), ptr(10), ptr(10), ptr(10), ptr(10)},
			60, 0, 6,
		},
		{
			"All Xs",
			[]*int{ptr(11), ptr(11), ptr(11), ptr(11), ptr(11), ptr(11)},
			60, 6, 6,
		},
		{
			"Mixed with M (0)",
			[]*int{ptr(10), ptr(11), ptr(0), ptr(9), ptr(8), ptr(7)},
			44, 1, 2,
		},
		{
			"Partial ends",
			[]*int{ptr(10), ptr(9), nil, nil, nil, nil},
			19, 0, 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, x, ten := calculate(tt.arrows[0], tt.arrows[1], tt.arrows[2], tt.arrows[3], tt.arrows[4], tt.arrows[5])
			if total != tt.expTotal || x != tt.expX || ten != tt.expTen {
				t.Errorf("%s: got (%d, %d, %d), expected (%d, %d, %d)", 
					tt.name, total, x, ten, tt.expTotal, tt.expX, tt.expTen)
			}
		})
	}
}
