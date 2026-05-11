package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractGlobalOptionsSupportsServerFlagAnywhere(t *testing.T) {
	options, args, err := extractGlobalOptions([]string{"login", "--server", "http://localhost:8080", "token"})
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080", options.Server)
	require.Equal(t, []string{"login", "token"}, args)
}

func TestExtractGlobalOptionsSupportsServerEquals(t *testing.T) {
	options, args, err := extractGlobalOptions([]string{"--server=http://localhost:8080", "."})
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080", options.Server)
	require.Equal(t, []string{"."}, args)
}
