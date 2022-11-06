package dfa

import (
	"fmt"
	"testing"
)

func TestNewSymbolPosition(t *testing.T) {
	tests := []struct {
		n       uint16
		endMark bool
		err     bool
	}{
		{
			n:       0,
			endMark: false,
			err:     true,
		},
		{
			n:       0,
			endMark: true,
			err:     true,
		},
		{
			n:       symbolPositionMin - 1,
			endMark: false,
			err:     true,
		},
		{
			n:       symbolPositionMin - 1,
			endMark: true,
			err:     true,
		},
		{
			n:       symbolPositionMin,
			endMark: false,
		},
		{
			n:       symbolPositionMin,
			endMark: true,
		},
		{
			n:       symbolPositionMax,
			endMark: false,
		},
		{
			n:       symbolPositionMax,
			endMark: true,
		},
		{
			n:       symbolPositionMax + 1,
			endMark: false,
			err:     true,
		},
		{
			n:       symbolPositionMax + 1,
			endMark: true,
			err:     true,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v n: %v, endMark: %v", i, tt.n, tt.endMark), func(t *testing.T) {
			pos, err := newSymbolPosition(tt.n, tt.endMark)
			if tt.err {
				if err == nil {
					t.Fatal("err is nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			n, endMark := pos.describe()
			if n != tt.n || endMark != tt.endMark {
				t.Errorf("unexpected symbol position: want: n: %v, endMark: %v, got: n: %v, endMark: %v", tt.n, tt.endMark, n, endMark)
			}
		})
	}
}
