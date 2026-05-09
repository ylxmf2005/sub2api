package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkstationAttachURL(t *testing.T) {
	got, err := WorkstationAttachURL("https://ccgo.example.com/base/", 42)
	require.NoError(t, err)
	require.Equal(t, "wss://ccgo.example.com/base/api/v1/ccgo/workstations/42/attach", got)

	got, err = WorkstationAttachURL("http://127.0.0.1:8080", 7)
	require.NoError(t, err)
	require.Equal(t, "ws://127.0.0.1:8080/api/v1/ccgo/workstations/7/attach", got)

	_, err = WorkstationAttachURL("https://ccgo.example.com", 0)
	require.ErrorContains(t, err, "workspace id")
}
