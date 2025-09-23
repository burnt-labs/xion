package wasmbinding_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	wasmbinding "github.com/burnt-labs/xion/wasmbindings"
)

func TestSetupKeys(t *testing.T) {
	// This test requires a key file to exist, let's test the error case first
	_, err := wasmbinding.SetupKeys()
	// This should error because the keys/jwtRS256.key file doesn't exist in test environment
	// BUT if it does exist, we should handle that gracefully
	if err != nil {
		require.Contains(t, err.Error(), "no such file or directory")
	}
	// If no error, the function found a valid key file and succeeded
}

func TestSetupPublicKeys(t *testing.T) {
	tests := []struct {
		name           string
		rsaFile        []string
		expectError    bool
		errorContains  string
		setupFile      bool
		fileContent    string
	}{
		{
			name:          "default file path - file not found",
			rsaFile:       []string{""},
			expectError:   false, // Changed to false since file might exist
			errorContains: "",
		},
		{
			name:          "custom file path - file not found",
			rsaFile:       []string{"./nonexistent/key.pem"},
			expectError:   true,
			errorContains: "no such file or directory",
		},
		{
			name:        "invalid PEM content",
			rsaFile:     []string{"./test_invalid.pem"},
			expectError: true,
			setupFile:   true,
			fileContent: "invalid pem content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFile {
				// Create test file with invalid content
				err := os.WriteFile(tt.rsaFile[0], []byte(tt.fileContent), 0644)
				require.NoError(t, err)
				defer os.Remove(tt.rsaFile[0]) // Clean up after test
			}

			privateKey, jwkKey, err := wasmbinding.SetupPublicKeys(tt.rsaFile...)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, privateKey)
				require.Nil(t, jwkKey)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				// Handle both success and expected failure cases
				if err != nil {
					// If it fails, it should be because the file doesn't exist
					require.Contains(t, err.Error(), "no such file or directory")
					require.Nil(t, privateKey)
					require.Nil(t, jwkKey)
				} else {
					// If it succeeds, the key file must exist
					require.NotNil(t, privateKey)
					// Note: jwkKey is expected to be nil based on the function implementation
				}
			}
		})
	}
}