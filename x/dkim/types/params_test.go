package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()

	require.NotNil(t, params)
	require.Equal(t, uint64(1), params.VkeyIdentifier)
	require.Equal(t, types.DefaultMaxPubKeySizeBytes, params.MaxPubkeySizeBytes)

	// Verify PublicInputIndices defaults
	indices := params.PublicInputIndices
	require.Equal(t, types.DefaultMinPublicInputsLength, indices.MinLength)
	require.Equal(t, types.DefaultEmailHashIndex, indices.EmailHashIndex)
	require.Equal(t, types.DefaultDkimDomainRangeStart, indices.DkimDomainRange.Start)
	require.Equal(t, types.DefaultDkimDomainRangeEnd, indices.DkimDomainRange.End)
	require.Equal(t, types.DefaultDkimHashIndex, indices.DkimHashIndex)
	require.Equal(t, types.DefaultTxBytesRangeStart, indices.TxBytesRange.Start)
	require.Equal(t, types.DefaultTxBytesRangeEnd, indices.TxBytesRange.End)
	require.Equal(t, types.DefaultEmailHostRangeStart, indices.EmailHostRange.Start)
	require.Equal(t, types.DefaultEmailHostRangeEnd, indices.EmailHostRange.End)
	require.Equal(t, types.DefaultEmailSubjectRangeStart, indices.EmailSubjectRange.Start)
	require.Equal(t, types.DefaultEmailSubjectRangeEnd, indices.EmailSubjectRange.End)
}

func TestParams_String(t *testing.T) {
	t.Run("default params to string", func(t *testing.T) {
		params := types.DefaultParams()
		str := params.String()

		require.NotEmpty(t, str)
		require.Contains(t, str, "vkey_identifier")
	})

	t.Run("empty params to string", func(t *testing.T) {
		params := types.Params{}
		str := params.String()

		require.NotEmpty(t, str)
		// Should be valid JSON even when empty
		require.Contains(t, str, "{")
		require.Contains(t, str, "}")
	})
}

func TestParams_Validate(t *testing.T) {
	t.Run("default params are valid", func(t *testing.T) {
		params := types.DefaultParams()
		err := params.Validate()
		require.NoError(t, err)
	})

	t.Run("params with valid custom indices are valid", func(t *testing.T) {
		params := types.DefaultParams()
		params.PublicInputIndices.MinLength = 100
		params.PublicInputIndices.EmailHashIndex = 90
		err := params.Validate()
		require.NoError(t, err)
	})

	t.Run("params with zero max_pubkey_size_bytes are invalid", func(t *testing.T) {
		params := types.DefaultParams()
		params.MaxPubkeySizeBytes = 0
		err := params.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "max_pubkey_size_bytes must be positive")
	})

	t.Run("params with zero vkey_identifier are invalid", func(t *testing.T) {
		params := types.DefaultParams()
		params.VkeyIdentifier = 0
		err := params.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "vkey_identifier must be positive")
	})

	t.Run("params with positive vkey_identifier are valid", func(t *testing.T) {
		params := types.DefaultParams()
		params.VkeyIdentifier = 42
		err := params.Validate()
		require.NoError(t, err)
	})
}

func TestIndexRange_Validate(t *testing.T) {
	t.Run("valid range", func(t *testing.T) {
		r := types.IndexRange{Start: 0, End: 9}
		err := r.Validate("test_range")
		require.NoError(t, err)
	})

	t.Run("invalid range - end equals start", func(t *testing.T) {
		r := types.IndexRange{Start: 5, End: 5}
		err := r.Validate("test_range")
		require.Error(t, err)
		require.Contains(t, err.Error(), "end (5) must be greater than start (5)")
	})

	t.Run("invalid range - end less than start", func(t *testing.T) {
		r := types.IndexRange{Start: 10, End: 5}
		err := r.Validate("test_range")
		require.Error(t, err)
		require.Contains(t, err.Error(), "end (5) must be greater than start (10)")
	})
}

func TestIndexRange_Length(t *testing.T) {
	r := types.IndexRange{Start: 12, End: 68}
	require.Equal(t, uint64(56), r.Length())
}

func TestPublicInputIndices_Validate(t *testing.T) {
	t.Run("default indices are valid", func(t *testing.T) {
		indices := types.DefaultPublicInputIndices()
		err := indices.Validate()
		require.NoError(t, err)
	})

	t.Run("zero min_length is invalid", func(t *testing.T) {
		indices := types.DefaultPublicInputIndices()
		indices.MinLength = 0
		err := indices.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "min_length must be positive")
	})

	t.Run("email_hash_index >= min_length is invalid", func(t *testing.T) {
		indices := types.DefaultPublicInputIndices()
		indices.EmailHashIndex = 88 // equal to min_length
		err := indices.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "email_hash_index (88) must be less than min_length (88)")
	})

	t.Run("dkim_hash_index >= min_length is invalid", func(t *testing.T) {
		indices := types.DefaultPublicInputIndices()
		indices.DkimHashIndex = 100
		err := indices.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "dkim_hash_index (100) must be less than min_length (88)")
	})

	t.Run("dkim_domain_range.end > min_length is invalid", func(t *testing.T) {
		indices := types.DefaultPublicInputIndices()
		indices.DkimDomainRange.End = 100
		err := indices.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "dkim_domain_range.end (100) must be <= min_length (88)")
	})

	t.Run("tx_bytes_range.end > min_length is invalid", func(t *testing.T) {
		indices := types.DefaultPublicInputIndices()
		indices.TxBytesRange.End = 100
		err := indices.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "tx_bytes_range.end (100) must be <= min_length (88)")
	})

	t.Run("email_host_range.end > min_length is invalid", func(t *testing.T) {
		indices := types.DefaultPublicInputIndices()
		indices.EmailHostRange.End = 100
		err := indices.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "email_host_range.end (100) must be <= min_length (88)")
	})

	t.Run("email_subject_range.end > min_length is invalid", func(t *testing.T) {
		indices := types.DefaultPublicInputIndices()
		indices.EmailSubjectRange.End = 100
		err := indices.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "email_subject_range.end (100) must be <= min_length (88)")
	})

	t.Run("invalid dkim_domain_range", func(t *testing.T) {
		indices := types.DefaultPublicInputIndices()
		indices.DkimDomainRange = types.IndexRange{Start: 10, End: 5}
		err := indices.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "dkim_domain_range: end (5) must be greater than start (10)")
	})

	t.Run("invalid tx_bytes_range", func(t *testing.T) {
		indices := types.DefaultPublicInputIndices()
		indices.TxBytesRange = types.IndexRange{Start: 50, End: 40}
		err := indices.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "tx_bytes_range: end (40) must be greater than start (50)")
	})

	t.Run("invalid email_host_range", func(t *testing.T) {
		indices := types.DefaultPublicInputIndices()
		indices.EmailHostRange = types.IndexRange{Start: 79, End: 70}
		err := indices.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "email_host_range: end (70) must be greater than start (79)")
	})

	t.Run("invalid email_subject_range", func(t *testing.T) {
		indices := types.DefaultPublicInputIndices()
		indices.EmailSubjectRange = types.IndexRange{Start: 88, End: 79}
		err := indices.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "email_subject_range: end (79) must be greater than start (88)")
	})
}

func TestDefaultPublicInputIndices(t *testing.T) {
	indices := types.DefaultPublicInputIndices()

	require.Equal(t, uint64(88), indices.MinLength)
	require.Equal(t, uint64(68), indices.EmailHashIndex)
	require.Equal(t, uint64(0), indices.DkimDomainRange.Start)
	require.Equal(t, uint64(9), indices.DkimDomainRange.End)
	require.Equal(t, uint64(9), indices.DkimHashIndex)
	require.Equal(t, uint64(12), indices.TxBytesRange.Start)
	require.Equal(t, uint64(68), indices.TxBytesRange.End)
	require.Equal(t, uint64(70), indices.EmailHostRange.Start)
	require.Equal(t, uint64(79), indices.EmailHostRange.End)
	require.Equal(t, uint64(79), indices.EmailSubjectRange.Start)
	require.Equal(t, uint64(88), indices.EmailSubjectRange.End)
}

func TestDefaultIndexRange(t *testing.T) {
	r := types.DefaultIndexRange(10, 20)
	require.Equal(t, uint64(10), r.Start)
	require.Equal(t, uint64(20), r.End)
}
