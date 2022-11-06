package compressor

import (
	"fmt"
	"testing"
)

func TestCompressor_Compress(t *testing.T) {
	x := 0 // an empty value

	allCompressors := func() []Compressor {
		return []Compressor{
			NewUniqueEntriesTable(),
			NewRowDisplacementTable(x),
		}
	}

	tests := []struct {
		original    []int
		rowCount    int
		colCount    int
		compressors []Compressor
	}{
		{
			original: []int{
				1, 1, 1, 1, 1,
				1, 1, 1, 1, 1,
				1, 1, 1, 1, 1,
			},
			rowCount:    3,
			colCount:    5,
			compressors: allCompressors(),
		},
		{
			original: []int{
				x, x, x, x, x,
				x, x, x, x, x,
				x, x, x, x, x,
			},
			rowCount:    3,
			colCount:    5,
			compressors: allCompressors(),
		},
		{
			original: []int{
				1, 1, 1, 1, 1,
				x, x, x, x, x,
				1, 1, 1, 1, 1,
			},
			rowCount:    3,
			colCount:    5,
			compressors: allCompressors(),
		},
		{
			original: []int{
				1, x, 1, 1, 1,
				1, 1, x, 1, 1,
				1, 1, 1, x, 1,
			},
			rowCount:    3,
			colCount:    5,
			compressors: allCompressors(),
		},
	}
	for i, tt := range tests {
		for _, comp := range tt.compressors {
			t.Run(fmt.Sprintf("%T #%v", comp, i), func(t *testing.T) {
				dup := make([]int, len(tt.original))
				copy(dup, tt.original)

				orig, err := NewOriginalTable(tt.original, tt.colCount)
				if err != nil {
					t.Fatal(err)
				}
				err = comp.Compress(orig)
				if err != nil {
					t.Fatal(err)
				}
				rowCount, colCount := comp.OriginalTableSize()
				if rowCount != tt.rowCount || colCount != tt.colCount {
					t.Fatalf("unexpected table size; want: %vx%v, got: %vx%v", tt.rowCount, tt.colCount, rowCount, colCount)
				}
				for i := 0; i < tt.rowCount; i++ {
					for j := 0; j < tt.colCount; j++ {
						v, err := comp.Lookup(i, j)
						if err != nil {
							t.Fatal(err)
						}
						expected := tt.original[i*tt.colCount+j]
						if v != expected {
							t.Fatalf("unexpected entry (%v, %v); want: %v, got: %v", i, j, expected, v)
						}
					}
				}

				// Calling with out-of-range indexes should be an error.
				if _, err := comp.Lookup(0, -1); err == nil {
					t.Fatalf("expected error didn't occur (0, -1)")
				}
				if _, err := comp.Lookup(-1, 0); err == nil {
					t.Fatalf("expected error didn't occur (-1, 0)")
				}
				if _, err := comp.Lookup(rowCount-1, colCount); err == nil {
					t.Fatalf("expected error didn't occur (%v, %v)", rowCount-1, colCount)
				}
				if _, err := comp.Lookup(rowCount, colCount-1); err == nil {
					t.Fatalf("expected error didn't occur (%v, %v)", rowCount, colCount-1)
				}

				// The compressor must not break the original table.
				for i := 0; i < tt.rowCount; i++ {
					for j := 0; j < tt.colCount; j++ {
						idx := i*tt.colCount + j
						if tt.original[idx] != dup[idx] {
							t.Fatalf("the original table is broken (%v, %v); want: %v, got: %v", i, j, dup[idx], tt.original[idx])
						}
					}
				}
			})
		}
	}
}
