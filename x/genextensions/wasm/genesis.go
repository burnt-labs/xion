package wasm

import (
	"context"
	"sync"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/burnt-labs/xion/x/genextensions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	WasmExportItemTypeParams   = 1
	WasmExportItemTypeCode     = 2
	WasmExportItemTypeContract = 3
	WasmExportItemTypeSequence = 4
)

type WasmGenesisExtension struct {
	keeper *wasmkeeper.Keeper
	cdc    codec.BinaryCodec

	codePool     *sync.Pool
	contractPool *sync.Pool

	contractAddrPool    *sync.Pool
	govPermissionKeeper *wasmkeeper.PermissionedKeeper
}

func NewWasmGenesisExtension(keeper *wasmkeeper.Keeper, cdc codec.BinaryCodec) *WasmGenesisExtension {
	return &WasmGenesisExtension{
		keeper: keeper,
		cdc:    cdc,
		codePool: &sync.Pool{
			New: func() any {
				return &wasmtypes.Code{}
			},
		},
		contractPool: &sync.Pool{
			New: func() any {
				return &wasmtypes.Contract{}
			},
		},
		contractAddrPool: &sync.Pool{
			New: func() any {
				return &sdk.AccAddress{}
			},
		},
		govPermissionKeeper: wasmkeeper.NewGovPermissionKeeper(keeper),
	}
}

func (e *WasmGenesisExtension) Export(ctx context.Context, export func(types.ExportItem) error) error {

	// export params
	params := e.keeper.GetParams(ctx)
	bz := e.cdc.MustMarshal(&params)
	err := export(types.ExportItem{
		Type:  WasmExportItemTypeParams,
		Value: bz,
	})
	if err != nil {
		return err
	}

	// export codes
	e.keeper.IterateCodeInfos(ctx, func(codeID uint64, codeInfo wasmtypes.CodeInfo) bool {
		bytecode, err := e.keeper.GetByteCode(ctx, codeID)
		if err != nil {
			panic(err)
		}
		code := wasmtypes.Code{
			CodeID:    codeID,
			CodeInfo:  codeInfo,
			CodeBytes: bytecode,
			Pinned:    e.keeper.IsPinnedCode(ctx, codeID),
		}
		bz := e.cdc.MustMarshal(&code)
		err = export(types.ExportItem{
			Type:  WasmExportItemTypeCode,
			Value: bz,
		})
		// if error, stop iterating
		return err != nil
	})

	// TODO: (found a few contracts with ~2GB of state)
	// export contract, original genesis export makes it in a single types.Contract object
	// but we will split it in 3 items to avoid writing big proto messages:
	// - contract info (types.ContractInfo)
	// - contract state (k,v)
	// - contract history ([]types.ContractHistoryEntry)

	// export contract in a single item (contract info, state, history)
	e.keeper.IterateContractInfo(ctx, func(addr sdk.AccAddress, contractInfo wasmtypes.ContractInfo) bool {
		var state []wasmtypes.Model
		e.keeper.IterateContractState(ctx, addr, func(key, value []byte) bool {
			state = append(state, wasmtypes.Model{Key: key, Value: value})
			return false
		})

		contractCodeHistory := e.keeper.GetContractHistory(ctx, addr)

		contract := wasmtypes.Contract{
			ContractAddress:     addr.String(),
			ContractInfo:        contractInfo,
			ContractState:       state,
			ContractCodeHistory: contractCodeHistory,
		}
		bz := e.cdc.MustMarshal(&contract)
		err = export(types.ExportItem{
			Type:  WasmExportItemTypeContract,
			Value: bz,
		})
		// if error, stop iterating
		return err != nil
	})

	for _, k := range [][]byte{wasmtypes.KeySequenceCodeID, wasmtypes.KeySequenceInstanceID} {
		id, err := e.keeper.PeekAutoIncrementID(ctx, k)
		if err != nil {
			panic(err)
		}
		bz := e.cdc.MustMarshal(&wasmtypes.Sequence{
			IDKey: k,
			Value: id,
		})
		err = export(types.ExportItem{
			Type:  WasmExportItemTypeSequence,
			Value: bz,
		})
		// if error, stop iterating
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *WasmGenesisExtension) Import(ctx sdk.Context, item *types.ExportItem) error {
	switch item.Type {
	case WasmExportItemTypeParams:
		var params wasmtypes.Params
		e.cdc.MustUnmarshal(item.Value, &params)
		e.keeper.SetParams(ctx, params)
	case WasmExportItemTypeCode:
		code := e.codePool.Get().(*wasmtypes.Code)
		defer e.codePool.Put(code)
		// reset the code object from the pool
		*code = wasmtypes.Code{}
		e.cdc.MustUnmarshal(item.Value, code)
		return e.importCode(ctx, code)
	case WasmExportItemTypeContract:
		contract := e.contractPool.Get().(*wasmtypes.Contract)
		defer e.contractPool.Put(contract)
		// reset the contract object from the pool
		*contract = wasmtypes.Contract{}
		e.cdc.MustUnmarshal(item.Value, contract)
		return e.importContract(ctx, contract)
	case WasmExportItemTypeSequence:
		var sequence wasmtypes.Sequence
		e.cdc.MustUnmarshal(item.Value, &sequence)
		return e.keeper.ImportAutoIncrementID(ctx, sequence.IDKey, sequence.Value)
	}
	return nil
}

func (e *WasmGenesisExtension) importCode(ctx sdk.Context, code *wasmtypes.Code) error {
	err := e.keeper.ImportCode(ctx, code.CodeID, code.CodeInfo, code.CodeBytes)
	if err != nil {
		return err
	}
	if code.Pinned {
		err = e.govPermissionKeeper.PinCode(ctx, code.CodeID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *WasmGenesisExtension) importContract(ctx sdk.Context, contract *wasmtypes.Contract) error {
	contractAddr, err := sdk.AccAddressFromBech32(contract.ContractAddress)
	if err != nil {
		return err
	}
	return e.keeper.ImportContract(ctx, contractAddr, &contract.ContractInfo, contract.ContractState, contract.ContractCodeHistory)
}
