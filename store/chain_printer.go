package store

import (
	"bytes"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"sort"
	"strconv"
	"strings"
)

// The bytes length of prefix hash to show
const hashLength = 3

type cbTable struct {
	// It's a fork path table. Every row is a fork, and every blocks in a column has same height. The first item in each row is the root block of the fork
	Rows [][]*CBlock
	// It is used to sort blocks in a column
	SortingColumn int
	// The width of block height string
	HeightWidth int
	// Hash of the head block of the current fork
	CurrentHeadHash common.Hash
}

// Dump forks into a table. Every row is a fork, and every blocks in a column has same height. The first item in each row is the root block of the fork
func newCbTable(leafBlocks []*CBlock, currentHash common.Hash) *cbTable {
	lineCount := len(leafBlocks)
	result := &cbTable{
		Rows:            make([][]*CBlock, lineCount),
		CurrentHeadHash: currentHash,
	}

	rowIndex := 0
	for _, leafNode := range leafBlocks {
		result.Rows[rowIndex] = reverse(leafNode.CollectToParent(nil))
		rowIndex++

		// find max height string width
		width := len(strconv.Itoa(int(leafNode.Block.Height())))
		if width > result.HeightWidth {
			result.HeightWidth = width
		}
	}

	return result
}

func (t cbTable) Len() int {
	return len(t.Rows)
}

func (t cbTable) Less(i, j int) bool {
	iBlock := t.Rows[i][t.SortingColumn]
	jBlock := t.Rows[j][t.SortingColumn]
	if iBlock == nil || jBlock == nil {
		log.Warn("Sorting error block")
		return false
	}
	iHash := iBlock.Block.Hash()
	jHash := jBlock.Block.Hash()
	return bytes.Compare(iHash[:], jHash[:]) < 0
}

func (t cbTable) Swap(i, j int) {
	t.Rows[i], t.Rows[j] = t.Rows[j], t.Rows[i]
}

// Sort sort the fork from startRow to endRow (not include endRow) by hash dictionary order at sortingColumn.
func (t cbTable) Sort(startRow, endRow, sortingColumn int) {
	rowCount := endRow - startRow
	if rowCount <= 1 {
		return
	}

	// sort from start row to end row
	sortTarget := &cbTable{
		Rows:          t.Rows[startRow:endRow],
		SortingColumn: sortingColumn,
	}
	sort.Sort(sortTarget)

	// sort recursively
	mergeStart := 0
	for i := 0; i < rowCount-1; i++ {
		block := sortTarget.Rows[i][sortingColumn]
		nextBlock := sortTarget.Rows[i+1][sortingColumn]
		// merge same blocks on different rows
		if block != nextBlock {
			// sort next column
			sortTarget.Sort(mergeStart, i+1, sortingColumn+1)
			mergeStart = i + 1
		} else {
			if i == rowCount-2 {
				sortTarget.Sort(mergeStart, rowCount, sortingColumn+1)
			}
		}
	}

	// concat the original rows and sorted rows
	rowsBefore := t.Rows[:startRow]
	rowsAfter := t.Rows[endRow:]
	t.Rows = append(append(rowsBefore, sortTarget.Rows...), rowsAfter...)
}

func isSameParent(rows [][]*CBlock, row1, row2, column int) bool {
	if row1 < 0 || row2 < 0 {
		return false
	}
	if len(rows[row1]) <= column || len(rows[row2]) <= column {
		return false
	}
	return rows[row1][column].Parent == rows[row2][column].Parent
}

func (t cbTable) String() string {
	type flagCell struct {
		connection int
		block      *CBlock
	}
	const (
		u = 0x1   // up
		d = 0x10  // down
		r = 0x100 // right
	)
	flagTable := make([][]flagCell, len(t.Rows))
	for i := 0; i < len(t.Rows); i++ {
		row := t.Rows[i]
		flagTable[i] = make([]flagCell, len(row))
		for j := 0; j < len(row); j++ {
			flagTable[i][j] = flagCell{}

			// only show every block once
			if i == 0 || len(t.Rows[i-1]) <= j || t.Rows[i-1][j] != t.Rows[i][j] {
				flagTable[i][j].connection = r
				flagTable[i][j].block = row[j]
			}

			// connect different up down neighbor if they had same parent
			if isSameParent(t.Rows, i-1, i, j) && t.Rows[i-1][j] != t.Rows[i][j] {
				flagTable[i-1][j].connection |= d
				flagTable[i][j].connection |= u
				// connect blocks in the column for the scene below
				// ├[101]1f5603d3┬[102]44ae7c5c
				// │             └[102]6490a06e
				// └[101]29d8a578-[102]379da997
				startRow := i - 1
				for isSameParent(t.Rows, startRow-1, startRow, j) {
					flagTable[startRow-1][j].connection |= d
					flagTable[startRow][j].connection |= u
					startRow--
				}
			}
		}
	}
	var sb strings.Builder
	for _, row := range flagTable {
		for _, cell := range row {
			switch cell.connection {
			case 0x0:
				sb.WriteString(" ")
			case 0x11:
				sb.WriteString("│")
			case 0x100:
				sb.WriteString("─")
			case 0x101:
				sb.WriteString("└")
			case 0x110:
				sb.WriteString("┬")
			case 0x111:
				sb.WriteString("├")
			default:
				log.Warn("unknown connection %x\n", cell.connection)
			}
			if cell.block != nil {
				hash := cell.block.Block.Hash()
				format := fmt.Sprintf("[%%%dd]%%x", t.HeightWidth)
				sb.WriteString(fmt.Sprintf(format, cell.block.Block.Height(), hash[:hashLength]))
				if cell.block.Block.Hash() == t.CurrentHeadHash {
					sb.WriteString(" <-Current")
				}
			} else {
				sb.WriteString(strings.Repeat(" ", 2+t.HeightWidth+hashLength*2))
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func reverse(src []*CBlock) []*CBlock {
	count := len(src)
	if count <= 1 {
		return src
	}

	result := make([]*CBlock, count)
	for i, item := range src {
		result[count-i-1] = item
	}
	return result
}

func SerializeForks(unconfirmedBlocks map[common.Hash]*CBlock, currentHash common.Hash) string {
	leaves := make([]*CBlock, 0, len(unconfirmedBlocks))
	for _, block := range unconfirmedBlocks {
		if len(block.Children) == 0 {
			leaves = append(leaves, block)
		}
	}
	if len(leaves) == 0 {
		return ""
	}

	table := newCbTable(leaves, currentHash)
	table.Sort(0, len(table.Rows), 0)
	return table.String()
}
