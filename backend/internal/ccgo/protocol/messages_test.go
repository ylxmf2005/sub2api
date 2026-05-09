package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestResponsePayloadRoundTrip(t *testing.T) {
	req, err := NewRequest("req-1", MethodFileStat, FileStatRequest{Path: "README.md"})
	require.NoError(t, err)
	require.Equal(t, MessageTypeRequest, req.Type)
	require.Equal(t, MethodFileStat, req.Method)

	decodedReq, err := DecodePayload[FileStatRequest](req)
	require.NoError(t, err)
	require.Equal(t, "README.md", decodedReq.Path)

	resp, err := NewResponse("req-1", FileReadResponse{Data: []byte("hello"), EOF: true})
	require.NoError(t, err)
	decodedResp, err := DecodePayload[FileReadResponse](resp)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), decodedResp.Data)
	require.True(t, decodedResp.EOF)
}
