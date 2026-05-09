package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type ccgoDeviceLoginServiceStub struct {
	startBaseURL string
	startDevice  string
	pollCode     string
	approveUser  int64
	approveCode  string

	startResult   *service.CcgoDeviceLoginStartResult
	pollResult    *service.CcgoDeviceLoginPollResult
	approveResult *service.CcgoDeviceLoginApproveResult
	startErr      error
	pollErr       error
	approveErr    error
}

func (s *ccgoDeviceLoginServiceStub) StartDeviceLogin(_ context.Context, serverBaseURL, deviceID string) (*service.CcgoDeviceLoginStartResult, error) {
	s.startBaseURL = serverBaseURL
	s.startDevice = deviceID
	if s.startErr != nil {
		return nil, s.startErr
	}
	return s.startResult, nil
}

func (s *ccgoDeviceLoginServiceStub) PollDeviceLogin(_ context.Context, deviceCode string) (*service.CcgoDeviceLoginPollResult, error) {
	s.pollCode = deviceCode
	if s.pollErr != nil {
		return nil, s.pollErr
	}
	return s.pollResult, nil
}

func (s *ccgoDeviceLoginServiceStub) ApproveDeviceLogin(_ context.Context, userID int64, userCode string) (*service.CcgoDeviceLoginApproveResult, error) {
	s.approveUser = userID
	s.approveCode = userCode
	if s.approveErr != nil {
		return nil, s.approveErr
	}
	return s.approveResult, nil
}

func TestCcgoHandlerStartDeviceLoginUsesForwardedFrontendBaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expiresAt := time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC)
	stub := &ccgoDeviceLoginServiceStub{startResult: &service.CcgoDeviceLoginStartResult{
		DeviceCode:      "device_code",
		UserCode:        "ABCD-EFGH",
		VerificationURI: "https://app.ccgo.test/ccgo/device?code=ABCD-EFGH",
		ExpiresAt:       expiresAt,
		IntervalSeconds: 2,
	}}
	handler := &CcgoHandler{deviceLoginService: stub}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/ccgo/device-login/start", bytes.NewBufferString(`{"device_id":"dev_123"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Forwarded-Proto", "https")
	c.Request.Header.Set("X-Forwarded-Host", "app.ccgo.test")

	handler.StartDeviceLogin(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "https://app.ccgo.test", stub.startBaseURL)
	require.Equal(t, "dev_123", stub.startDevice)
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			UserCode string `json:"user_code"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Equal(t, "ABCD-EFGH", envelope.Data.UserCode)
}

func TestCcgoHandlerPollDeviceLoginSurfacesPendingReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &ccgoDeviceLoginServiceStub{pollErr: service.ErrCcgoDeviceLoginPending}
	handler := &CcgoHandler{deviceLoginService: stub}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/ccgo/device-login/poll", bytes.NewBufferString(`{"device_code":"device_code"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.PollDeviceLogin(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, "device_code", stub.pollCode)
	var envelope struct {
		Reason string `json:"reason"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, "CCGO_DEVICE_LOGIN_PENDING", envelope.Reason)
}

func TestCcgoHandlerApproveDeviceLoginRequiresAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &ccgoDeviceLoginServiceStub{approveResult: &service.CcgoDeviceLoginApproveResult{
		Status:    service.CcgoDeviceLoginStatusApproved,
		UserID:    7,
		ExpiresAt: time.Now().Add(time.Minute),
	}}
	handler := &CcgoHandler{deviceLoginService: stub}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/ccgo/device-login/approve", bytes.NewBufferString(`{"user_code":"ABCD-EFGH"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ApproveDeviceLogin(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(7), stub.approveUser)
	require.Equal(t, "ABCD-EFGH", stub.approveCode)
}
