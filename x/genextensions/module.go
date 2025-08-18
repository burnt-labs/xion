package genextensions

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"cosmossdk.io/core/address"
	feegrant "cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	authzgenesisextension "github.com/burnt-labs/xion/x/genextensions/authz"
	feegrantgenesisextension "github.com/burnt-labs/xion/x/genextensions/feegrant"
	"github.com/burnt-labs/xion/x/genextensions/types"
	wasmgenesisextension "github.com/burnt-labs/xion/x/genextensions/wasm"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/protoio"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/klauspost/compress/zstd"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
)

const ModuleName = "genextensions"

var (
	_ module.AppModuleGenesis = AppModule{}
	_ module.AppModuleBasic   = AppModuleBasic{}
)

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string {
	return ModuleName
}

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
}

func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	// TODO: this is a hack to get the abstract account genesis account to work
	// we need to upstream this to the abstract account module
	registry.RegisterImplementations(
		(*authtypes.GenesisAccount)(nil),
		&aatypes.AbstractAccount{},
	)
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
}

type AppModule struct {
	AppModuleBasic
	genesisDir string
	addrCodec  address.Codec

	authzExtension    *authzgenesisextension.AuthzGenesisExtension
	feegrantExtension *feegrantgenesisextension.FeeGrantGenesisExtension
	wasmExtension     *wasmgenesisextension.WasmGenesisExtension
}

func NewAppModule(genesisDir string, authzKeeper authzkeeper.Keeper, fgk feegrantkeeper.Keeper,
	bankKeeper feegrant.BankKeeper, wasmKeeper *wasmkeeper.Keeper, cdc codec.BinaryCodec, addrCodec address.Codec) module.GenesisOnlyAppModule {
	if addrCodec == nil {
		panic("addrCodec is nil")
	}
	feeGrantKeeper := fgk.SetBankKeeper(bankKeeper)

	return module.NewGenesisOnlyAppModule(AppModule{
		genesisDir: genesisDir,
		addrCodec:  addrCodec,
		authzExtension: authzgenesisextension.NewAuthzGenesisExtension(
			authzKeeper,
			cdc,
			addrCodec,
		),
		feegrantExtension: feegrantgenesisextension.NewFeeGrantGenesisExtension(
			feeGrantKeeper,
			cdc,
			addrCodec,
		),
		wasmExtension: wasmgenesisextension.NewWasmGenesisExtension(
			wasmKeeper,
			cdc,
		),
	})
}

func (AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genesis := types.GenesisState{
		Modules: make(map[string]types.ModuleExport),
	}

	bytes, err := json.Marshal(genesis)
	if err != nil {
		panic(err)
	}
	return bytes
}

func (AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genesis types.GenesisState
	if err := json.Unmarshal(bz, &genesis); err != nil {
		return err
	}
	return nil
}

func (a AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	importOrder := []string{authz.ModuleName, feegrant.ModuleName, wasmtypes.ModuleName}
	for _, module := range importOrder {
		fmt.Println("importing module", module)
		switch module {
		case authz.ModuleName:
			m, ok := genesisState.Modules[module]
			if !ok {
				continue
			}

			importedItems, err := a.importExtension(ctx, path.Join(a.genesisDir, m.File.Path), a.authzExtension)
			if err != nil {
				panic(err)
			}
			if importedItems != m.File.TotalItems {
				panic(fmt.Errorf("imported items %d != %d", importedItems, m.File.TotalItems))
			}
			fmt.Println("imported authz items", importedItems)
		case feegrant.ModuleName:
			m, ok := genesisState.Modules[module]
			if !ok {
				continue
			}

			importedItems, err := a.importExtension(ctx, path.Join(a.genesisDir, m.File.Path), a.feegrantExtension)
			if err != nil {
				panic(err)
			}
			if importedItems != m.File.TotalItems {
				panic(fmt.Errorf("imported items %d != %d", importedItems, m.File.TotalItems))
			}
			fmt.Println("imported feegrant items", importedItems)
		case wasmtypes.ModuleName:
			m, ok := genesisState.Modules[module]
			if !ok {
				continue
			}

			importedItems, err := a.importExtension(ctx, path.Join(a.genesisDir, m.File.Path), a.wasmExtension)
			if err != nil {
				panic(err)
			}
			if importedItems != m.File.TotalItems {
				panic(fmt.Errorf("imported items %d != %d", importedItems, m.File.TotalItems))
			}
			fmt.Println("imported wasm items", importedItems)
		}
	}

	return nil
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func (a AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	exportDir := path.Join(a.genesisDir, "export")
	err := ensureDir(exportDir)
	if err != nil {
		panic(err)
	}

	genesis := types.GenesisState{
		Modules: make(map[string]types.ModuleExport),
	}

	genesis.Modules[feegrant.ModuleName] = a.export(ctx, exportDir, fmt.Sprintf("%s.pb.zst", feegrant.ModuleName), a.feegrantExtension)
	genesis.Modules[authz.ModuleName] = a.export(ctx, exportDir, fmt.Sprintf("%s.pb.zst", authz.ModuleName), a.authzExtension)
	genesis.Modules[wasmtypes.ModuleName] = a.export(ctx, exportDir, fmt.Sprintf("%s.pb.zst", wasmtypes.ModuleName), a.wasmExtension)

	return cdc.MustMarshalJSON(&genesis)
}

type Exporter interface {
	Export(ctx context.Context, export func(types.ExportItem) error) error
}

// export performs an extension genesis export by iterating through all the items from the extension implementation
// all items are piped from protoio.DelimitedWriter-> Zstd Compression ->  Buffer & Hash Writer -> File
// Buffer & Hash Writer is a multi writer so that the hash is computed on the final compressed file on the fly
func (a AppModule) export(ctx sdk.Context, exportDir string, fileName string, extension Exporter) types.ModuleExport {
	exportFile, err := os.OpenFile(path.Join(exportDir, fileName), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer func() {
		fmt.Println("closing export file")
		if closeErr := exportFile.Close(); closeErr != nil {
			fmt.Printf("Error closing export file: %v\n", closeErr)
			panic(closeErr)
		}
	}()

	bufw := bufio.NewWriterSize(exportFile, 1024*1024*4) // 4MB buffer

	filehash := sha256.New()

	// buf with hash writer (zstd will pipe through this hash will be computed on the final compressed file)
	bufwhWriter := io.MultiWriter(bufw, filehash)

	zstdWriter, err := zstd.NewWriter(bufwhWriter)
	if err != nil {
		panic(err)
	}

	writer := protoio.NewDelimitedWriter(zstdWriter)

	totalItems := uint64(0)
	err = extension.Export(ctx, func(item types.ExportItem) error {
		if _, writeErr := writer.WriteMsg(&item); writeErr != nil {
			return fmt.Errorf("failed to write message: %w", writeErr)
		}
		totalItems++
		return nil
	})
	if err != nil {
		panic(err)
	}
	// finalize stream before computing checksum
	if err := writer.Close(); err != nil {
		panic(fmt.Errorf("close delimited writer: %w", err))
	}
	if err := zstdWriter.Close(); err != nil {
		panic(fmt.Errorf("close zstd: %w", err))
	}
	if err := bufw.Flush(); err != nil {
		panic(fmt.Errorf("flush bufio: %w", err))
	}

	checksum := filehash.Sum(nil)

	return types.ModuleExport{
		Format:  "proto+delimited+zstd",
		Version: "v1",
		File: types.File{
			Path:       fileName,
			Checksum:   hex.EncodeToString(checksum),
			TotalItems: totalItems,
		},
	}
}

type Importer interface {
	Import(ctx sdk.Context, item *types.ExportItem) error
}

func (a AppModule) importExtension(ctx sdk.Context, fileName string, importer Importer) (uint64, error) {
	importFile, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		return 0, err
	}
	defer importFile.Close()

	bufReader := bufio.NewReaderSize(importFile, 1024*1024*128) // 128MB buffer
	zstdReader, err := zstd.NewReader(bufReader)
	if err != nil {
		return 0, err
	}
	defer zstdReader.Close()

	// Currently read up to 3GB per item size until contract state is split into multiple files
	reader := protoio.NewDelimitedReader(zstdReader, 3*1024*1024*1024) // 3GB max per item size

	defer reader.Close()

	importedItems := uint64(0)
	for {
		item := &types.ExportItem{}
		_, err := reader.ReadMsg(item)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return importedItems, err
		}
		err = importer.Import(ctx, item)
		if err != nil {
			return importedItems, err
		}
		importedItems++

	}
	return importedItems, nil
}
