package testlib

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLatestGithubReleaseSendsToken(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"tag_name":"v31.0.2","name":"v31.0.2"}`))
	}))
	defer srv.Close()

	tag, err := getLatestGithubRelease(srv.Client(), srv.URL, "burnt-labs/xion", "tok")
	require.NoError(t, err)
	require.Equal(t, "v31.0.2", tag)
	require.Equal(t, "Bearer tok", gotAuth)
	require.Equal(t, "/repos/burnt-labs/xion/releases/latest", gotPath)
}

func TestGetLatestGithubReleaseWithoutToken(t *testing.T) {
	sawAuth := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, sawAuth = r.Header["Authorization"]
		_, _ = w.Write([]byte(`{"tag_name":"v31.0.2"}`))
	}))
	defer srv.Close()

	_, err := getLatestGithubRelease(srv.Client(), srv.URL, "burnt-labs/xion", "")
	require.NoError(t, err)
	require.False(t, sawAuth, "no Authorization header without a token")
}

func TestGetLatestGithubReleaseReportsRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := getLatestGithubRelease(srv.Client(), srv.URL, "burnt-labs/xion", "")
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "403") && strings.Contains(err.Error(), "rate limit remaining 0"), err.Error())
}
