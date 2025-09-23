package main

import (
	"sync"

	"github.com/spf13/cobra"

	"github.com/burnt-labs/xion/app/params"
)

// Shared test utilities to avoid config sealing issues
var (
	testRootCmd        *cobra.Command
	testEncodingConfig params.EncodingConfig
	setupOnce          sync.Once
)

func setupTestEnvironment() {
	setupOnce.Do(func() {
		testRootCmd, testEncodingConfig = NewRootCmd()
	})
}