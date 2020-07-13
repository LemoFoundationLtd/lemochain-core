package consensus

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"

	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/network"
)

func TestNewConfirmer(t *testing.T) {
	bLoader := &testBlockLoader{}
	dm := deputynode.NewManager(5, bLoader)
	sLoader := createTestBlockStore(testBlocks[1])
	confirmer := NewConfirmer(dm, bLoader, nil, sLoader)

	assert.Equal(t, bLoader, confirmer.blockLoader)
	assert.Equal(t, sLoader, confirmer.stableLoader)
	assert.Equal(t, dm, confirmer.dm)
	assert.Equal(t, testBlocks[1].Height(), confirmer.lastSig.Height)
	assert.Equal(t, testBlocks[1].Hash(), confirmer.lastSig.Hash)
}

func TestConfirmer_TryConfirm(t *testing.T) {
	type fields struct {
		blockLoader  BlockLoader
		stableLoader StableBlockStore
		confirmStore confirmWriter
		dm           *deputynode.Manager
		lastSig      blockSignRecord
	}
	type args struct {
		block *types.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   types.SignData
		want1  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmer{
				blockLoader:  tt.fields.blockLoader,
				stableLoader: tt.fields.stableLoader,
				confirmStore: tt.fields.confirmStore,
				dm:           tt.fields.dm,
				lastSig:      tt.fields.lastSig,
			}
			got, got1 := c.TryConfirm(tt.args.block)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Confirmer.TryConfirm() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Confirmer.TryConfirm() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestConfirmer_needConfirm(t *testing.T) {
	type fields struct {
		blockLoader  BlockLoader
		stableLoader StableBlockStore
		confirmStore confirmWriter
		dm           *deputynode.Manager
		lastSig      blockSignRecord
	}
	type args struct {
		block *types.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmer{
				blockLoader:  tt.fields.blockLoader,
				stableLoader: tt.fields.stableLoader,
				confirmStore: tt.fields.confirmStore,
				dm:           tt.fields.dm,
				lastSig:      tt.fields.lastSig,
			}
			if got := c.needConfirm(tt.args.block); got != tt.want {
				t.Errorf("Confirmer.needConfirm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmer_BatchConfirmStable(t *testing.T) {
	type fields struct {
		blockLoader  BlockLoader
		stableLoader StableBlockStore
		confirmStore confirmWriter
		dm           *deputynode.Manager
		lastSig      blockSignRecord
	}
	type args struct {
		startHeight uint32
		endHeight   uint32
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*network.BlockConfirmData
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmer{
				blockLoader:  tt.fields.blockLoader,
				stableLoader: tt.fields.stableLoader,
				confirmStore: tt.fields.confirmStore,
				dm:           tt.fields.dm,
				lastSig:      tt.fields.lastSig,
			}
			if got := c.BatchConfirmStable(tt.args.startHeight, tt.args.endHeight); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Confirmer.BatchConfirmStable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmer_NeedConfirmList(t *testing.T) {
	type fields struct {
		blockLoader  BlockLoader
		stableLoader StableBlockStore
		confirmStore confirmWriter
		dm           *deputynode.Manager
		lastSig      blockSignRecord
	}
	type args struct {
		startHeight uint32
		endHeight   uint32
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []network.GetConfirmInfo
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmer{
				blockLoader:  tt.fields.blockLoader,
				stableLoader: tt.fields.stableLoader,
				confirmStore: tt.fields.confirmStore,
				dm:           tt.fields.dm,
				lastSig:      tt.fields.lastSig,
			}
			if got := c.NeedConfirmList(tt.args.startHeight, tt.args.endHeight); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Confirmer.NeedConfirmList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmer_SetLastSig(t *testing.T) {
	type fields struct {
		blockLoader  BlockLoader
		stableLoader StableBlockStore
		confirmStore confirmWriter
		dm           *deputynode.Manager
		lastSig      blockSignRecord
	}
	type args struct {
		block *types.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmer{
				blockLoader:  tt.fields.blockLoader,
				stableLoader: tt.fields.stableLoader,
				confirmStore: tt.fields.confirmStore,
				dm:           tt.fields.dm,
				lastSig:      tt.fields.lastSig,
			}
			c.SetLastSig(tt.args.block)
		})
	}
}

func TestIsMinedByself(t *testing.T) {
	type args struct {
		block *types.Block
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMinedByself(tt.args.block); got != tt.want {
				t.Errorf("IsMinedByself() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmer_tryConfirmStable(t *testing.T) {
	type fields struct {
		blockLoader  BlockLoader
		stableLoader StableBlockStore
		confirmStore confirmWriter
		dm           *deputynode.Manager
		lastSig      blockSignRecord
	}
	type args struct {
		block *types.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *types.SignData
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmer{
				blockLoader:  tt.fields.blockLoader,
				stableLoader: tt.fields.stableLoader,
				confirmStore: tt.fields.confirmStore,
				dm:           tt.fields.dm,
				lastSig:      tt.fields.lastSig,
			}
			if got := c.tryConfirmStable(tt.args.block); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Confirmer.tryConfirmStable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmer_SaveConfirm(t *testing.T) {
	type fields struct {
		blockLoader  BlockLoader
		stableLoader StableBlockStore
		confirmStore confirmWriter
		dm           *deputynode.Manager
		lastSig      blockSignRecord
	}
	type args struct {
		block   *types.Block
		sigList []types.SignData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.Block
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmer{
				blockLoader:  tt.fields.blockLoader,
				stableLoader: tt.fields.stableLoader,
				confirmStore: tt.fields.confirmStore,
				dm:           tt.fields.dm,
				lastSig:      tt.fields.lastSig,
			}
			got, err := c.SaveConfirm(tt.args.block, tt.args.sigList)
			if (err != nil) != tt.wantErr {
				t.Errorf("Confirmer.SaveConfirm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Confirmer.SaveConfirm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmer_confirmBlock(t *testing.T) {
	type fields struct {
		blockLoader  BlockLoader
		stableLoader StableBlockStore
		confirmStore confirmWriter
		dm           *deputynode.Manager
		lastSig      blockSignRecord
	}
	type args struct {
		block *types.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    types.SignData
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confirmer{
				blockLoader:  tt.fields.blockLoader,
				stableLoader: tt.fields.stableLoader,
				confirmStore: tt.fields.confirmStore,
				dm:           tt.fields.dm,
				lastSig:      tt.fields.lastSig,
			}
			got, err := c.confirmBlock(tt.args.block)
			if (err != nil) != tt.wantErr {
				t.Errorf("Confirmer.confirmBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Confirmer.confirmBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}
