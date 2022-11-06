package utf8

import (
	"fmt"
	"testing"
)

func TestGenCharBlocks_WellFormed(t *testing.T) {
	cBlk := func(from []byte, to []byte) *CharBlock {
		return &CharBlock{
			From: from,
			To:   to,
		}
	}

	seq := func(b ...byte) []byte {
		return b
	}

	tests := []struct {
		from   rune
		to     rune
		blocks []*CharBlock
	}{
		{
			from: '\u0000',
			to:   '\u007f',
			blocks: []*CharBlock{
				cBlk(seq(0x00), seq(0x7f)),
			},
		},
		{
			from: '\u0080',
			to:   '\u07ff',
			blocks: []*CharBlock{
				cBlk(seq(0xc2, 0x80), seq(0xdf, 0xbf)),
			},
		},
		{
			from: '\u0800',
			to:   '\u0fff',
			blocks: []*CharBlock{
				cBlk(seq(0xe0, 0xa0, 0x80), seq(0xe0, 0xbf, 0xbf)),
			},
		},
		{
			from: '\u1000',
			to:   '\ucfff',
			blocks: []*CharBlock{
				cBlk(seq(0xe1, 0x80, 0x80), seq(0xec, 0xbf, 0xbf)),
			},
		},
		{
			from: '\ud000',
			to:   '\ud7ff',
			blocks: []*CharBlock{
				cBlk(seq(0xed, 0x80, 0x80), seq(0xed, 0x9f, 0xbf)),
			},
		},
		{
			from: '\ue000',
			to:   '\uffff',
			blocks: []*CharBlock{
				cBlk(seq(0xee, 0x80, 0x80), seq(0xef, 0xbf, 0xbf)),
			},
		},
		{
			from: '\U00010000',
			to:   '\U0003ffff',
			blocks: []*CharBlock{
				cBlk(seq(0xf0, 0x90, 0x80, 0x80), seq(0xf0, 0xbf, 0xbf, 0xbf)),
			},
		},
		{
			from: '\U00040000',
			to:   '\U000fffff',
			blocks: []*CharBlock{
				cBlk(seq(0xf1, 0x80, 0x80, 0x80), seq(0xf3, 0xbf, 0xbf, 0xbf)),
			},
		},
		{
			from: '\U00100000',
			to:   '\U0010ffff',
			blocks: []*CharBlock{
				cBlk(seq(0xf4, 0x80, 0x80, 0x80), seq(0xf4, 0x8f, 0xbf, 0xbf)),
			},
		},
		{
			from: '\u0000',
			to:   '\U0010ffff',
			blocks: []*CharBlock{
				cBlk(seq(0x00), seq(0x7f)),
				cBlk(seq(0xc2, 0x80), seq(0xdf, 0xbf)),
				cBlk(seq(0xe0, 0xa0, 0x80), seq(0xe0, 0xbf, 0xbf)),
				cBlk(seq(0xe1, 0x80, 0x80), seq(0xec, 0xbf, 0xbf)),
				cBlk(seq(0xed, 0x80, 0x80), seq(0xed, 0x9f, 0xbf)),
				cBlk(seq(0xee, 0x80, 0x80), seq(0xef, 0xbf, 0xbf)),
				cBlk(seq(0xf0, 0x90, 0x80, 0x80), seq(0xf0, 0xbf, 0xbf, 0xbf)),
				cBlk(seq(0xf1, 0x80, 0x80, 0x80), seq(0xf3, 0xbf, 0xbf, 0xbf)),
				cBlk(seq(0xf4, 0x80, 0x80, 0x80), seq(0xf4, 0x8f, 0xbf, 0xbf)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v..%v", tt.from, tt.to), func(t *testing.T) {
			blks, err := GenCharBlocks(tt.from, tt.to)
			if err != nil {
				t.Fatal(err)
			}
			if len(blks) != len(tt.blocks) {
				t.Fatalf("unexpected character block: want: %+v, got: %+v", tt.blocks, blks)
			}
			for i, blk := range blks {
				if len(blk.From) != len(tt.blocks[i].From) || len(blk.To) != len(tt.blocks[i].To) {
					t.Fatalf("unexpected character block: want: %+v, got: %+v", tt.blocks, blks)
				}
				for j := 0; j < len(blk.From); j++ {
					if blk.From[j] != tt.blocks[i].From[j] || blk.To[j] != tt.blocks[i].To[j] {
						t.Fatalf("unexpected character block: want: %+v, got: %+v", tt.blocks, blks)
					}
				}
			}
		})
	}
}

func TestGenCharBlocks_IllFormed(t *testing.T) {
	tests := []struct {
		from rune
		to   rune
	}{
		{
			// from > to
			from: '\u0001',
			to:   '\u0000',
		},
		{
			from: -1, // <U+0000
			to:   '\u0000',
		},
		{
			from: '\u0000',
			to:   -1, // <U+0000
		},
		{
			from: 0x110000, // >U+10FFFF
			to:   '\u0000',
		},
		{
			from: '\u0000',
			to:   0x110000, // >U+10FFFF
		},
		{
			from: 0xd800, // U+D800 (surrogate code point)
			to:   '\ue000',
		},
		{
			from: 0xdfff, // U+DFFF (surrogate code point)
			to:   '\ue000',
		},
		{
			from: '\ucfff',
			to:   0xd800, // U+D800 (surrogate code point)
		},
		{
			from: '\ucfff',
			to:   0xdfff, // U+DFFF (surrogate code point)
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v..%v", tt.from, tt.to), func(t *testing.T) {
			blks, err := GenCharBlocks(tt.from, tt.to)
			if err == nil {
				t.Fatal("expected error didn't occur")
			}
			if blks != nil {
				t.Fatal("character blocks must be nil")
			}
		})
	}
}
