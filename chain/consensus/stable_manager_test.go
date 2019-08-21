package consensus

import (
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/store"
	"github.com/stretchr/testify/assert"
	"reflect"
	"sync"
	"testing"

	"github.com/LemoFoundationLtd/lemochain-core/chain/deputynode"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
)

// testBlockStore is a block list. The last of it is latest block. If set any block to stable, it will remove the previous blocks as prune
type testBlockStore []*types.Block

func (bs testBlockStore) LoadLatestBlock() (*types.Block, error) {
	if len(bs) == 0 {
		return nil, store.ErrNotExist
	}
	return bs[len(bs)-1], nil
}

// SetStableBlock set a block to stable, then remove the previous blocks as prune
func (bs testBlockStore) SetStableBlock(hash common.Hash) ([]*types.Block, error) {
	for i := 0; i < len(bs); i++ {
		if bs[i].Hash() == hash {
			pruned := bs[:i]
			bs = bs[i:]
			return pruned, nil
		}
	}
	return nil, store.ErrArgInvalid
}

func TestNewStableManager(t *testing.T) {
	dm := deputynode.NewManager(5, testBlockLoader{})
	sm := NewStableManager(dm, testBlockStore{})
	assert.Equal(t, dm, sm.dm)
}

func TestStableManager_StableBlock(t *testing.T) {
	dm := deputynode.NewManager(5, testBlockLoader{})

	// empty
	sm := NewStableManager(dm, testBlockStore{})
	assert.PanicsWithValue(t, store.ErrNotExist, func() {
		sm.StableBlock()
	})

	// not empty
	sm = NewStableManager(dm, testBlockStore{testBlocks[0]})
	block := sm.StableBlock()
	assert.Equal(t, testBlocks[0], block)
}

func TestStableManager_UpdateStable(t *testing.T) {
	type fields struct {
		store StableBlockStore
		dm    *deputynode.Manager
		lock  sync.Mutex
	}
	type args struct {
		block *types.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		want1   []*types.Block
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StableManager{
				store: tt.fields.store,
				dm:    tt.fields.dm,
				lock:  tt.fields.lock,
			}
			got, got1, err := sm.UpdateStable(tt.args.block)
			if (err != nil) != tt.wantErr {
				t.Errorf("StableManager.UpdateStable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("StableManager.UpdateStable() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("StableManager.UpdateStable() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestIsConfirmEnough(t *testing.T) {
	var tests = []struct {
		ConfirmCount int
		DeputyCount  int
		Want         bool
	}{
		{0, 0, true},
		{0, 1, false},
		{1, 1, true},
		{0, 2, false},
		{1, 2, false},
		{2, 2, true},
	}

	for _, test := range tests {
		caseName := fmt.Sprintf("confirm %d/%d", test.ConfirmCount, test.DeputyCount)
		t.Run(caseName, func(t *testing.T) {
			test := test // capture range variable
			t.Parallel()

			dm := deputynode.NewManager(test.DeputyCount, testBlockLoader{})
			block := testBlocks[0].Copy()
			block.Confirms = make([]types.SignData, test.ConfirmCount)
			assert.Equal(t, test.Want, IsConfirmEnough(block, dm))
		})
	}
}
