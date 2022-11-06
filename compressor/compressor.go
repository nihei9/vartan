package compressor

import (
	"encoding/binary"
	"fmt"
	"sort"
)

type OriginalTable struct {
	entries  []int
	rowCount int
	colCount int
}

func NewOriginalTable(entries []int, colCount int) (*OriginalTable, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("enries is empty")
	}
	if colCount <= 0 {
		return nil, fmt.Errorf("colCount must be >=1")
	}
	if len(entries)%colCount != 0 {
		return nil, fmt.Errorf("entries length or column count are incorrect; entries length: %v, column count: %v", len(entries), colCount)
	}

	return &OriginalTable{
		entries:  entries,
		rowCount: len(entries) / colCount,
		colCount: colCount,
	}, nil
}

type Compressor interface {
	Compress(orig *OriginalTable) error
	Lookup(row, col int) (int, error)
	OriginalTableSize() (int, int)
}

var (
	_ Compressor = &UniqueEntriesTable{}
	_ Compressor = &RowDisplacementTable{}
)

type UniqueEntriesTable struct {
	UniqueEntries    []int
	RowNums          []int
	OriginalRowCount int
	OriginalColCount int
}

func NewUniqueEntriesTable() *UniqueEntriesTable {
	return &UniqueEntriesTable{}
}

func (tab *UniqueEntriesTable) Lookup(row, col int) (int, error) {
	if row < 0 || row >= tab.OriginalRowCount || col < 0 || col >= tab.OriginalColCount {
		return 0, fmt.Errorf("indexes are out of range: [%v, %v]", row, col)
	}
	return tab.UniqueEntries[tab.RowNums[row]*tab.OriginalColCount+col], nil
}

func (tab *UniqueEntriesTable) OriginalTableSize() (int, int) {
	return tab.OriginalRowCount, tab.OriginalColCount
}

func (tab *UniqueEntriesTable) Compress(orig *OriginalTable) error {
	var uniqueEntries []int
	rowNums := make([]int, orig.rowCount)
	hash2RowNum := map[string]int{}
	nextRowNum := 0
	for row := 0; row < orig.rowCount; row++ {
		var rowHash string
		{
			buf := make([]byte, 0, orig.colCount*8)
			for col := 0; col < orig.colCount; col++ {
				b := make([]byte, 8)
				binary.PutUvarint(b, uint64(orig.entries[row*orig.colCount+col]))
				buf = append(buf, b...)
			}
			rowHash = string(buf)
		}
		rowNum, ok := hash2RowNum[rowHash]
		if !ok {
			rowNum = nextRowNum
			nextRowNum++
			hash2RowNum[rowHash] = rowNum
			start := row * orig.colCount
			entry := append([]int{}, orig.entries[start:start+orig.colCount]...)
			uniqueEntries = append(uniqueEntries, entry...)
		}
		rowNums[row] = rowNum
	}

	tab.UniqueEntries = uniqueEntries
	tab.RowNums = rowNums
	tab.OriginalRowCount = orig.rowCount
	tab.OriginalColCount = orig.colCount

	return nil
}

const ForbiddenValue = -1

type RowDisplacementTable struct {
	OriginalRowCount int
	OriginalColCount int
	EmptyValue       int
	Entries          []int
	Bounds           []int
	RowDisplacement  []int
}

func NewRowDisplacementTable(emptyValue int) *RowDisplacementTable {
	return &RowDisplacementTable{
		EmptyValue: emptyValue,
	}
}

func (tab *RowDisplacementTable) Lookup(row int, col int) (int, error) {
	if row < 0 || row >= tab.OriginalRowCount || col < 0 || col >= tab.OriginalColCount {
		return tab.EmptyValue, fmt.Errorf("indexes are out of range: [%v, %v]", row, col)
	}
	d := tab.RowDisplacement[row]
	if tab.Bounds[d+col] != row {
		return tab.EmptyValue, nil
	}
	return tab.Entries[d+col], nil
}

func (tab *RowDisplacementTable) OriginalTableSize() (int, int) {
	return tab.OriginalRowCount, tab.OriginalColCount
}

type rowInfo struct {
	rowNum        int
	nonEmptyCount int
	nonEmptyCol   []int
}

func (tab *RowDisplacementTable) Compress(orig *OriginalTable) error {
	rowInfo := make([]rowInfo, orig.rowCount)
	{
		row := 0
		col := 0
		rowInfo[0].rowNum = 0
		for _, v := range orig.entries {
			if col == orig.colCount {
				row++
				col = 0
				rowInfo[row].rowNum = row
			}
			if v != tab.EmptyValue {
				rowInfo[row].nonEmptyCount++
				rowInfo[row].nonEmptyCol = append(rowInfo[row].nonEmptyCol, col)
			}
			col++
		}

		sort.SliceStable(rowInfo, func(i int, j int) bool {
			return rowInfo[i].nonEmptyCount > rowInfo[j].nonEmptyCount
		})
	}

	origEntriesLen := len(orig.entries)
	entries := make([]int, origEntriesLen)
	bounds := make([]int, origEntriesLen)
	resultBottom := orig.colCount
	rowDisplacement := make([]int, orig.rowCount)
	{
		for i := 0; i < origEntriesLen; i++ {
			entries[i] = tab.EmptyValue
			bounds[i] = ForbiddenValue
		}

		nextRowDisplacement := 0
		for _, rInfo := range rowInfo {
			if rInfo.nonEmptyCount <= 0 {
				continue
			}

			for {
				isOverlapped := false
				for _, col := range rInfo.nonEmptyCol {
					if entries[nextRowDisplacement+col] == tab.EmptyValue {
						continue
					}
					nextRowDisplacement++
					isOverlapped = true
					break
				}
				if isOverlapped {
					continue
				}

				rowDisplacement[rInfo.rowNum] = nextRowDisplacement
				for _, col := range rInfo.nonEmptyCol {
					entries[nextRowDisplacement+col] = orig.entries[(rInfo.rowNum*orig.colCount)+col]
					bounds[nextRowDisplacement+col] = rInfo.rowNum
				}
				resultBottom = nextRowDisplacement + orig.colCount
				nextRowDisplacement++
				break
			}
		}
	}

	tab.OriginalRowCount = orig.rowCount
	tab.OriginalColCount = orig.colCount
	tab.Entries = entries[:resultBottom]
	tab.Bounds = bounds[:resultBottom]
	tab.RowDisplacement = rowDisplacement

	return nil
}
