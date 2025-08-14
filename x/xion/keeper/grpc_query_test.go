package keeper

import (
	"bytes"
	"context"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestPanicParseCredentialBadRequestResponseBody(t *testing.T) {
	jsonBodyCreate := []byte(`{"id":"UWxY-yRdIls8IT-vyMS6la1ZiqESOAff7bWZ_LWV0Pg","type":"public-key","rawId":"VVd4WS15UmRJbHM4SVQtdnlNUzZsYTFaaXFFU09BZmY3YldaX0xXVjBQZw","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uY3JlYXRlIiwiY2hhbGxlbmdlIjoiZUdsdmJqRjZjbXcxTm5jMGMyVjNkMlJ4ZW1keGVHYzNOemMzYkRWbFptWjVlSHBsTUcweVkyc3llVGh6TUdGdVpITnpNRzVoTUdkeE1uSmxhRE5qIiwib3JpZ2luIjoiaHR0cHM6Ly94aW9uLWRhcHAtZXhhbXBsZS1naXQtZmVhdC1mYWNlaWQtYnVybnRmaW5hbmNlLnZlcmNlbC5hcHAifQ","attestationObject":"o2NmbXRkbm9uZWhBdXRoRGF0YaVkcnBpZFggsGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw1lZmxhZ3MYRWhhdHRfZGF0YaNmYWFndWlkUEFBR1VJREFBR1VJREFBPT1qcHVibGljX2tleVkCEKQBAwM5AQAgWQIAolg7TF3aai-wR4HTDe5oR-WRhEsdW3u-O3IJHl0BiHkmR4MLskHG9HzivWoXsloUBnBMrFNxOH0x5cNMI07oi4PeRbHySiogRW9CXPjJaNlTi-pT_IgKFsyJNXsLyzrnajLkDbQU6pRsHmNeL0hAOUv48rtXv8VVWWN8okJehD2q9N7LHoFAOmIUEPg_VTHTt8K__O-9eMZKN4eMjh_4-sxRX6NXPSPT87XRlrK4GZ4pUdp86K0tOFLhwO4Uj0JkMNfI82eVZ1tAbDlqjd8jFnAb8fWm8wtdaTNbL_AAXmbDhswwJOyrw8fARZIhrXSdKBWa6e4k7sLwTIy-OO8saebnlARsjGst7ZCzmw5KCm2ctEVl3hYhHwyXu_A5rOblMrV3H0G7WqeKMCMVSJ11ssrlsmfVhNIwu1Qlt5GYmPTTJiCgGUGRxZkgDyOyjFNHglYpZamCGyJ9oyofsukEGoqMQ6WzjFi_hjVapzXi7Li-Q0OjEopIUUDDgeUrgjbGY0eiHI6sAz5hoaD0Qjc9e3Hk6-y7VcKCTCAanZOlJV0vJkHB98LBLh9qAoVUei_VaLFe2IcfVlrL_43aXlsHhr_SUQY5pHPlUMbQihE_57dpPRh31qDX_w6ye8dilniP8JmpKM2uIwnJ0x7hfJ45Qa0oLHmrGlzY9wi-RGP0YUkhQwEAAW1jcmVkZW50aWFsX2lkWCtVV3hZLXlSZElsczhJVC12eU1TNmxhMVppcUVTT0FmZjdiV1pfTFdWMFBnaGV4dF9kYXRh9mpzaWduX2NvdW50AGhhdXRoRGF0YVkCcrBjAYg3BKaYjH8UNdEzwntvhWiqy3k5L6c8XM54EzMNRQAAAABBQUdVSURBQUdVSURBQT09ACtVV3hZLXlSZElsczhJVC12eU1TNmxhMVppcUVTT0FmZjdiV1pfTFdWMFBnpAEDAzkBACBZAgCiWDtMXdpqL7BHgdMN7mhH5ZGESx1be747cgkeXQGIeSZHgwuyQcb0fOK9aheyWhQGcEysU3E4fTHlw0wjTuiLg95FsfJKKiBFb0Jc-Mlo2VOL6lP8iAoWzIk1ewvLOudqMuQNtBTqlGweY14vSEA5S_jyu1e_xVVZY3yiQl6EPar03ssegUA6YhQQ-D9VMdO3wr_87714xko3h4yOH_j6zFFfo1c9I9PztdGWsrgZnilR2nzorS04UuHA7hSPQmQw18jzZ5VnW0BsOWqN3yMWcBvx9abzC11pM1sv8ABeZsOGzDAk7KvDx8BFkiGtdJ0oFZrp7iTuwvBMjL447yxp5ueUBGyMay3tkLObDkoKbZy0RWXeFiEfDJe78Dms5uUytXcfQbtap4owIxVInXWyyuWyZ9WE0jC7VCW3kZiY9NMmIKAZQZHFmSAPI7KMU0eCVillqYIbIn2jKh-y6QQaioxDpbOMWL-GNVqnNeLsuL5DQ6MSikhRQMOB5SuCNsZjR6IcjqwDPmGhoPRCNz17ceTr7LtVwoJMIBqdk6UlXS8mQcH3wsEuH2oChVR6L9VosV7Yhx9WWsv_jdpeWweGv9JRBjmkc-VQxtCKET_nt2k9GHfWoNf_DrJ7x2KWeI_wmakoza4jCcnTHuF8njlBrSgseasaXNj3CL5EY_RhSSFDAQAB"}}`)

	credentials := bytes.NewReader(jsonBodyCreate)
	err := validateCredentialCreation(credentials)
	require.NoError(t, err)

	jsonBodyRequest := []byte(`{"id":"UWxY-yRdIls8IT-vyMS6la1ZiqESOAff7bWZ_LWV0Pg","type":"public-key","rawId":"VVd4WS15UmRJbHM4SVQtdnlNUzZsYTFaaXFFU09BZmY3YldaX0xXVjBQZw","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uZ2V0IiwiY2hhbGxlbmdlIjoiWkhVMVFsQnNNVUZvVXpKa05rVTVjMk0xTUcxTVdHMVBibVpxVjFaTlNtSkRZVlJzU25ZeE5GZHhaejAiLCJvcmlnaW4iOiJodHRwczovL3hpb24tZGFwcC1leGFtcGxlLWdpdC1mZWF0LWZhY2VpZC1idXJudGZpbmFuY2UudmVyY2VsLmFwcCJ9","authenticatorData":"sGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw1FAAAAAEFBR1VJREFBR1VJREFBPT0AK1VXeFkteVJkSWxzOElULXZ5TVM2bGExWmlxRVNPQWZmN2JXWl9MV1YwUGekAQMDOQEAIFkCAKJYO0xd2movsEeB0w3uaEflkYRLHVt7vjtyCR5dAYh5JkeDC7JBxvR84r1qF7JaFAZwTKxTcTh9MeXDTCNO6IuD3kWx8koqIEVvQlz4yWjZU4vqU_yIChbMiTV7C8s652oy5A20FOqUbB5jXi9IQDlL-PK7V7_FVVljfKJCXoQ9qvTeyx6BQDpiFBD4P1Ux07fCv_zvvXjGSjeHjI4f-PrMUV-jVz0j0_O10ZayuBmeKVHafOitLThS4cDuFI9CZDDXyPNnlWdbQGw5ao3fIxZwG_H1pvMLXWkzWy_wAF5mw4bMMCTsq8PHwEWSIa10nSgVmunuJO7C8EyMvjjvLGnm55QEbIxrLe2Qs5sOSgptnLRFZd4WIR8Ml7vwOazm5TK1dx9Bu1qnijAjFUiddbLK5bJn1YTSMLtUJbeRmJj00yYgoBlBkcWZIA8jsoxTR4JWKWWpghsifaMqH7LpBBqKjEOls4xYv4Y1Wqc14uy4vkNDoxKKSFFAw4HlK4I2xmNHohyOrAM-YaGg9EI3PXtx5Ovsu1XCgkwgGp2TpSVdLyZBwffCwS4fagKFVHov1WixXtiHH1Zay_-N2l5bB4a_0lEGOaRz5VDG0IoRP-e3aT0Yd9ag1_8OsnvHYpZ4j_CZqSjNriMJydMe4XyeOUGtKCx5qxpc2PcIvkRj9GFJIUMBAAE","signature":"OWJd1g5KplaPvqYt9Lv_dbR6NzqCVYi2bAWX6J5Dl_b9TW183AssulkgXwmVj0KHtlWkUjDOFsmIyeOMGo2BlQbtkB4b3G97CR0NtVNXMT3CJojIkB4xkegZti-rOLHUZbj0bZ1LOphmuqYEcO4ipZIAiB86VdeSht9_2xA8th3kuTwF6mRrm02ulmhyPWImrbODoQj-mhO3b-2HLdD64Desk0kRGNmw-YvixyXr4gzwH9jKwdaXOh4pzntqlt5fDZbwW1j70w4j7Q1dijnYFxvCS_51K-NMZJA5R8YbEq21NxxXEZLC9b2_4C944ehkAQ1DFDouQbXxCuualvLXSnOy89kgZmALzyu0gKXdhAuV6uS1oj9u8Ohsj09_xHd7IVKPAlstJabsP2eR5Q4LT6zkmz2njURw3NrChAMn2yiGdB1v09L_d3zQzjnVjP2Ki7HSILwh79lt0ejTSxh1VkqbLG82DYi2_3lomBXZszvpOLr7t_mfx92_0WTBFBw24glWEfBmEHUw6TS24YD3pQ30lMIHk9mk2m1CQjF_Wfew_4s2zk4uW-pOhD90FtERPSOKZeayyF8cTfIrV66HZ2pCnOcjLNNJ6wlA95N6rbtoKPLQS9Wkw38TF2D8aobZk3lDH4lYU2R4ODHwJhuKX_6R_0Y7CnBk9w5AbYbUJj0","userHandle":"eGlvbjF6cmw1Nnc0c2V3d2RxemdxeGc3Nzc3bDVlZmZ5eHplMG0yY2syeThzMGFuZHNzMG5hMGdxMnJlaDNj"}}`)
	err = validateCredentialRequest(bytes.NewReader(jsonBodyRequest))
	require.NoError(t, err)
}

func TestValidateCredentialCreation_ErrorPaths(t *testing.T) {
	tests := map[string]struct {
		input       string
		expectError bool
	}{
		"invalid JSON": {
			input:       `{"invalid": json}`,
			expectError: true,
		},
		"empty body": {
			input:       ``,
			expectError: true,
		},
		"missing attestationResponse": {
			input:       `{"id":"test"}`,
			expectError: true,
		},
		"invalid clientDataJSON": {
			input:       `{"attestationResponse":{"clientDataJSON":"invalid-base64","attestationObject":"o2NmbXRkbm9uZWhBdXRoRGF0YaVkcnBpZFggsGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw1lZmxhZ3MYRWhhdHRfZGF0YaNmYWFndWlkUEFBR1VJREFBR1VJREFBPT1qcHVibGljX2tleVkCEKQBAwM5AQAgWQIAolg7TF3aai-wR4HTDe5oR-WRhEsdW3u-O3IJHl0BiHkmR4MLskHG9HzivWoXsloUBnBMrFNxOH0x5cNMI07oi4PeRbHySiogRW9CXPjJaNlTi-pT_IgKFsyJNXsLyzrnajLkDbQU6pRsHmNeL0hAOUv48rtXv8VVWWN8okJehD2q9N7LHoFAOmIUEPg_VTHTt8K__O-9eMZKN4eMjh_4-sxRX6NXPSPT87XRlrK4GZ4pUdp86K0tOFLhwO4Uj0JkMNfI82eVZ1tAbDlqjd8jFnAb8fWm8wtdaTNbL_AAXmbDhswwJOyrw8fARZIhrXSdKBWa6e4k7sLwTIy-OO8saebnlARsjGst7ZCzmw5KCm2ctEVl3hYhHwyXu_A5rOblMrV3H0G7WqeKMCMVSJ11ssrlsmfVhNIwu1Qlt5GYmPTTJiCgGUGRxZkgDyOyjFNHglYpZamCGyJ9oyofsukEGoqMQ6WzjFi_hjVapzXi7Li-Q0OjEopIUUDDgeUrgjbGY0eiHI6sAz5hoaD0Qjc9e3Hk6-y7VcKCTCAanZOlJV0vJkHB98LBLh9qAoVUei_VaLFe2IcfVlrL_43aXlsHhr_SUQY5pHPlUMbQihE_57dpPRh31qDX_w6ye8dilniP8JmpKM2uIwnJ0x7hfJ45Qa0oLHmrGlzY9wi-RGP0YUkhQwEAAW1jcmVkZW50aWFsX2lkWCtVV3hZLXlSZElsczhJVC12eU1TNmxhMVppcUVTT0FmZjdiV1pfTFdWMFBnaGV4dF9kYXRh9mpzaWduX2NvdW50AGhhdXRoRGF0YVkCcrBjAYg3BKaYjH8UNdEzwntvhWiqy3k5L6c8XM54EzMNRQAAAABBQUdVSURBQUdVSURBQT09ACtVV3hZLXlSZElsczhJVC12eU1TNmxhMVppcUVTT0FmZjdiV1pfTFdWMFBnpAEDAzkBACBZAgCiWDtMXdpqL7BHgdMN7mhH5ZGESx1be747cgkeXQGIeSZHgwuyQcb0fOK9aheyWhQGcEysU3E4fTHlw0wjTuiLg95FsfJKKiBFb0Jc-Mlo2VOL6lP8iAoWzIk1ewvLOudqMuQNtBTqlGweY14vSEA5S_jyu1e_xVVZY3yiQl6EPar03ssegUA6YhQQ-D9VMdO3wr_87714xko3h4yOH_j6zFFfo1c9I9PztdGWsrgZnilR2nzorS04UuHA7hSPQmQw18jzZ5VnW0BsOWqN3yMWcBvx9abzC11pM1sv8ABeZsOGzDAk7KvDx8BFkiGtdJ0oFZrp7iTuwvBMjL447yxp5ueUBGyMay3tkLObDkoKbZy0RWXeFiEfDJe78Dms5uUytXcfQbtap4owIxVInXWyyuWyZ9WE0jC7VCW3kZiY9NMmIKAZQZHFmSAPI7KMU0eCVillqYIbIn2jKh-y6QQaioxDpbOMWL-GNVqnNeLsuL5DQ6MSikhRQMOB5SuCNsZjR6IcjqwDPmGhoPRCNz17ceTr7LtVwoJMIBqdk6UlXS8mQcH3wsEuH2oChVR6L9VosV7Yhx9WWsv_jdpeWweGv9JRBjmkc-VQxtCKET_nt2k9GHfWoNf_DrJ7x2KWeI_wmakoza4jCcnTHuF8njlBrSgseasaXNj3CL5EY_RhSSFDAQAB"}}`,
			expectError: true,
		},
		"invalid attestationObject": {
			input:       `{"attestationResponse":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uY3JlYXRlIiwiY2hhbGxlbmdlIjoiZUdsdmJqRjZjbXcxTm5jMGMyVjNkMlJ4ZW1keGVHYzNOemMzYkRWbFptWjVlSHBsTUcweVkyc3llVGh6TUdGdVpITnpNRzVoTUdkeE1uSmxhRE5qIiwib3JpZ2luIjoiaHR0cHM6Ly94aW9uLWRhcHAtZXhhbXBsZS1naXQtZmVhdC1mYWNlaWQtYnVybnRmaW5hbmNlLnZlcmNlbC5hcHAifQ","attestationObject":"invalid-cbor"}}`,
			expectError: true,
		},
		"valid JSON but short auth data": {
			input:       `{"attestationResponse":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uY3JlYXRlIn0","attestationObject":"o2NmbXRkbm9uZWhBdXRoRGF0YVggc2hvcnREYXRhDGZsYWdzFGhhdHRfZGF0YQ"}}`,
			expectError: true,
		},
		"CBOR with short auth data": {
			// Create a valid CBOR structure but with auth data that's too short
			input:       `{"attestationResponse":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uY3JlYXRlIn0","attestationObject":"o2NmbXRkbm9uZWhBdXRoRGF0YVABYW1hbHNob3J0"}}`,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateCredentialCreation(bytes.NewReader([]byte(tc.input)))
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateCredentialRequest_ErrorPaths(t *testing.T) {
	tests := map[string]struct {
		input       string
		expectError bool
	}{
		"invalid JSON": {
			input:       `{"invalid": json}`,
			expectError: true,
		},
		"empty body": {
			input:       ``,
			expectError: true,
		},
		"missing response": {
			input:       `{"id":"test"}`,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateCredentialRequest(bytes.NewReader([]byte(tc.input)))
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateAttestation_ErrorPaths(t *testing.T) {
	tests := map[string]struct {
		name        string
		rawAuthData []byte
		expectError bool
		errorMsg    string
	}{
		"too short data": {
			rawAuthData: make([]byte, 30), // Less than required 37 bytes
			expectError: true,
			errorMsg:    "expected data greater than 37 bytes",
		},
		"minimal valid data": {
			rawAuthData: make([]byte, 37), // Exactly 37 bytes
			expectError: false,
		},
		"data with extensions flag set": {
			rawAuthData: func() []byte {
				data := make([]byte, 37)
				data[32] = 0x80 // Set extensions flag (bit 7)
				return data
			}(),
			expectError: false,
		},
		"malformed data with extensions": {
			rawAuthData: func() []byte {
				data := make([]byte, 40) // 37 + 3 extra bytes
				data[32] = 0x80          // Set extensions flag
				return data
			}(),
			expectError: false,
		},
		"specific malformed case": {
			rawAuthData: func() []byte {
				// Try to create a case where len(rawAuthData)-remaining > len(rawAuthData)
				// This is mathematically impossible with the current logic, but let's try
				data := make([]byte, 37)
				data[32] = 0x80 // Set extensions flag
				return data
			}(),
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAttestation(tc.rawAuthData)
			if tc.expectError {
				require.Error(t, err)
				if tc.errorMsg != "" {
					require.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func setupQueryTest(t *testing.T) (context.Context, Keeper) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx

	keeper := Keeper{
		storeKey: key,
	}

	return sdk.WrapSDKContext(ctx), keeper
}

func TestKeeper_WebAuthNVerifyRegister_InvalidURL(t *testing.T) {
	ctx, keeper := setupQueryTest(t)

	request := &types.QueryWebAuthNVerifyRegisterRequest{
		Rp:        "://invalid-url",
		Addr:      "test_address",
		Challenge: "test_challenge",
		Data:      []byte(`{"id":"test"}`),
	}

	response, err := keeper.WebAuthNVerifyRegister(ctx, request)

	require.Error(t, err)
	require.Nil(t, response)
	require.Contains(t, err.Error(), "missing protocol scheme")
}

func TestKeeper_WebAuthNVerifyRegister_InvalidCredentialData(t *testing.T) {
	ctx, keeper := setupQueryTest(t)

	request := &types.QueryWebAuthNVerifyRegisterRequest{
		Rp:        "https://example.com",
		Addr:      "test_address",
		Challenge: "test_challenge",
		Data:      []byte(`{"invalid": "data"}`),
	}

	response, err := keeper.WebAuthNVerifyRegister(ctx, request)

	require.Error(t, err)
	require.Nil(t, response)
	require.Contains(t, err.Error(), "Web auth is not valid")
}

func TestKeeper_WebAuthNVerifyRegister_InvalidJSON(t *testing.T) {
	ctx, keeper := setupQueryTest(t)

	request := &types.QueryWebAuthNVerifyRegisterRequest{
		Rp:        "https://example.com",
		Addr:      "test_address",
		Challenge: "test_challenge",
		Data:      []byte(`invalid json`),
	}

	response, err := keeper.WebAuthNVerifyRegister(ctx, request)

	require.Error(t, err)
	require.Nil(t, response)
	require.Contains(t, err.Error(), "Web auth is not valid")
}

func TestKeeper_WebAuthNVerifyAuthenticate_InvalidURL(t *testing.T) {
	ctx, keeper := setupQueryTest(t)

	request := &types.QueryWebAuthNVerifyAuthenticateRequest{
		Rp:         "://invalid-url",
		Addr:       "test_address",
		Challenge:  "test_challenge",
		Credential: []byte(`{"test": "credential"}`),
		Data:       []byte(`{"id":"test"}`),
	}

	response, err := keeper.WebAuthNVerifyAuthenticate(ctx, request)

	require.Error(t, err)
	require.Nil(t, response)
	require.Contains(t, err.Error(), "missing protocol scheme")
}

func TestKeeper_WebAuthNVerifyAuthenticate_InvalidCredentialData(t *testing.T) {
	ctx, keeper := setupQueryTest(t)

	request := &types.QueryWebAuthNVerifyAuthenticateRequest{
		Rp:         "https://example.com",
		Addr:       "test_address",
		Challenge:  "test_challenge",
		Credential: []byte(`{"test": "credential"}`),
		Data:       []byte(`{"invalid": "data"}`),
	}

	response, err := keeper.WebAuthNVerifyAuthenticate(ctx, request)

	require.Error(t, err)
	require.Nil(t, response)
	require.Contains(t, err.Error(), "Web auth is not valid")
}

func TestKeeper_WebAuthNVerifyAuthenticate_InvalidCredentialJSON(t *testing.T) {
	ctx, keeper := setupQueryTest(t)

	request := &types.QueryWebAuthNVerifyAuthenticateRequest{
		Rp:         "https://example.com",
		Addr:       "test_address",
		Challenge:  "test_challenge",
		Credential: []byte(`invalid json`),
		Data:       []byte(`{"response":{"clientDataJSON":"test","authenticatorData":"test","signature":"test"}}`),
	}

	response, err := keeper.WebAuthNVerifyAuthenticate(ctx, request)

	require.Error(t, err)
	require.Nil(t, response)
	require.Contains(t, err.Error(), "Web auth is not valid")
}
