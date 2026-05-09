package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/hub"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/stretchr/testify/require"
)

type ccgoRepoStub struct {
	workspace   *CcgoWorkspace
	activeRun   *CcgoWorkstationRun
	latestRun   *CcgoWorkstationRun
	created     *CcgoWorkstationRun
	running     *CcgoWorkstationRun
	stopped     *CcgoWorkstationRun
	deviceLogin *CcgoDeviceLogin
}

func (r *ccgoRepoStub) CreateDeviceLogin(_ context.Context, deviceCodeHash, userCodeHash, deviceID string, expiresAt time.Time) (*CcgoDeviceLogin, error) {
	r.deviceLogin = &CcgoDeviceLogin{
		ID:             1,
		DeviceCodeHash: deviceCodeHash,
		UserCodeHash:   userCodeHash,
		DeviceID:       deviceID,
		Status:         CcgoDeviceLoginStatusPending,
		ExpiresAt:      expiresAt,
	}
	return r.deviceLogin, nil
}

func (r *ccgoRepoStub) FindDeviceLoginByDeviceCodeHash(_ context.Context, deviceCodeHash string) (*CcgoDeviceLogin, error) {
	if r.deviceLogin == nil || r.deviceLogin.DeviceCodeHash != deviceCodeHash {
		return nil, ErrCcgoDeviceLoginNotFound
	}
	return r.deviceLogin, nil
}

func (r *ccgoRepoStub) FindDeviceLoginByUserCodeHash(_ context.Context, userCodeHash string) (*CcgoDeviceLogin, error) {
	if r.deviceLogin == nil || r.deviceLogin.UserCodeHash != userCodeHash {
		return nil, ErrCcgoDeviceLoginNotFound
	}
	return r.deviceLogin, nil
}

func (r *ccgoRepoStub) ApproveDeviceLogin(_ context.Context, id int64, userID int64, now time.Time) (*CcgoDeviceLogin, error) {
	if r.deviceLogin == nil || r.deviceLogin.ID != id {
		return nil, ErrCcgoDeviceLoginNotFound
	}
	r.deviceLogin.Status = CcgoDeviceLoginStatusApproved
	r.deviceLogin.UserID = &userID
	r.deviceLogin.ApprovedAt = &now
	return r.deviceLogin, nil
}

func (r *ccgoRepoStub) ConsumeDeviceLogin(_ context.Context, id int64, now time.Time) (*CcgoDeviceLogin, error) {
	if r.deviceLogin == nil || r.deviceLogin.ID != id {
		return nil, ErrCcgoDeviceLoginNotFound
	}
	r.deviceLogin.Status = CcgoDeviceLoginStatusConsumed
	r.deviceLogin.ConsumedAt = &now
	return r.deviceLogin, nil
}

func (r *ccgoRepoStub) ExpireDeviceLogin(_ context.Context, id int64, _ time.Time) (*CcgoDeviceLogin, error) {
	if r.deviceLogin == nil || r.deviceLogin.ID != id {
		return nil, ErrCcgoDeviceLoginNotFound
	}
	r.deviceLogin.Status = CcgoDeviceLoginStatusExpired
	return r.deviceLogin, nil
}

func (r *ccgoRepoStub) ResolveWorkspace(context.Context, CcgoResolveWorkspaceInput) (*CcgoWorkspaceResolution, error) {
	return nil, ErrCcgoWorkspaceUnavailable
}

func (r *ccgoRepoStub) IssueAgentCredential(context.Context, *CcgoWorkspace, time.Duration) (*CcgoIssuedCredential, error) {
	return nil, ErrCcgoWorkspaceUnavailable
}

func (r *ccgoRepoStub) FindAgentCredentialByTokenHash(context.Context, string) (*CcgoAgentCredential, error) {
	return nil, ErrCcgoAgentCredentialInvalid
}

func (r *ccgoRepoStub) MarkAgentCredentialUsed(context.Context, int64) error {
	return ErrCcgoAgentCredentialInvalid
}

func (r *ccgoRepoStub) GetWorkspace(context.Context, int64) (*CcgoWorkspace, error) {
	if r.workspace == nil {
		return nil, ErrCcgoWorkspaceNotFound
	}
	return r.workspace, nil
}

func (r *ccgoRepoStub) FindActiveWorkstationRun(context.Context, int64) (*CcgoWorkstationRun, error) {
	return r.activeRun, nil
}

func (r *ccgoRepoStub) FindLatestWorkstationRun(context.Context, int64) (*CcgoWorkstationRun, error) {
	if r.latestRun != nil {
		return r.latestRun, nil
	}
	return r.activeRun, nil
}

func (r *ccgoRepoStub) CreateWorkstationRun(context.Context, *CcgoWorkspace, string, time.Time) (*CcgoWorkstationRun, error) {
	if r.created == nil {
		r.created = &CcgoWorkstationRun{WorkspaceID: r.workspace.ID, UserID: r.workspace.UserID, RunID: "run_created", Status: CcgoRunStatusStarting}
	}
	return r.created, nil
}

func (r *ccgoRepoStub) MarkWorkstationRunRunning(context.Context, string, string, time.Time) (*CcgoWorkstationRun, error) {
	if r.running == nil {
		r.running = &CcgoWorkstationRun{WorkspaceID: r.workspace.ID, UserID: r.workspace.UserID, RunID: "run_created", Status: CcgoRunStatusRunning}
	}
	return r.running, nil
}

func (r *ccgoRepoStub) MarkWorkstationRunStopped(context.Context, string, string, time.Time) (*CcgoWorkstationRun, error) {
	if r.stopped == nil {
		r.stopped = &CcgoWorkstationRun{WorkspaceID: r.workspace.ID, UserID: r.workspace.UserID, RunID: "run_created", Status: CcgoRunStatusStopped}
	}
	return r.stopped, nil
}

func (r *ccgoRepoStub) MarkWorkstationRunFailed(context.Context, string, string, time.Time) error {
	return nil
}

func (r *ccgoRepoStub) CreateCommandAudit(context.Context, CcgoCommandAuditInput) error {
	return nil
}

type noopTransport struct{}

func (noopTransport) Send(context.Context, protocol.Envelope) error {
	return nil
}

func (noopTransport) Recv(context.Context) (protocol.Envelope, error) {
	<-make(chan struct{})
	return protocol.Envelope{}, nil
}

func (noopTransport) Close() error {
	return nil
}

type runStarterStub struct {
	run    *CcgoWorkstationRun
	reused bool
	status *CcgoRunnerStatus
	stop   *CcgoWorkstationRun
}

type deviceLoginUserReaderStub struct {
	user *User
	err  error
}

func (s deviceLoginUserReaderStub) GetByID(context.Context, int64) (*User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

type deviceLoginTokenIssuerStub struct {
	pair        *TokenPair
	pairErr     error
	token       string
	expiresIn   int
	recordedIDs []int64
}

func (s *deviceLoginTokenIssuerStub) GenerateTokenPair(context.Context, *User, string) (*TokenPair, error) {
	if s.pairErr != nil {
		return nil, s.pairErr
	}
	return s.pair, nil
}

func (s *deviceLoginTokenIssuerStub) GenerateToken(*User) (string, error) {
	if s.token == "" {
		return "", errors.New("token not configured")
	}
	return s.token, nil
}

func (s *deviceLoginTokenIssuerStub) GetAccessTokenExpiresIn() int {
	return s.expiresIn
}

func (s *deviceLoginTokenIssuerStub) RecordSuccessfulLogin(_ context.Context, userID int64) {
	s.recordedIDs = append(s.recordedIDs, userID)
}

func (s runStarterStub) StartCcgoRun(context.Context, *CcgoWorkspace) (*CcgoWorkstationRun, bool, error) {
	return s.run, s.reused, nil
}

func (s runStarterStub) Attach(int64, io.Reader, io.Writer) error {
	return nil
}

func (s runStarterStub) RuntimeStatus(context.Context, int64) (*CcgoRunnerStatus, error) {
	return s.status, nil
}

func (s runStarterStub) StopCcgoRun(context.Context, int64, string) (*CcgoWorkstationRun, *CcgoRunnerStatus, error) {
	return s.stop, s.status, nil
}

func TestCcgoServiceStartWorkstationRequiresAgentConnection(t *testing.T) {
	svc := NewCcgoService(&ccgoRepoStub{workspace: &CcgoWorkspace{ID: 42, UserID: 7}}, hub.NewConnectionManager(), nil, nil, nil)

	_, err := svc.StartWorkstation(context.Background(), CcgoStartWorkstationInput{UserID: 7, WorkspaceID: 42})
	require.ErrorIs(t, err, ErrCcgoAgentDisconnected)
}

func TestCcgoServiceStartWorkstationCreatesRunWhenAgentConnected(t *testing.T) {
	manager := hub.NewConnectionManager()
	manager.Register(42, noopTransport{})
	repo := &ccgoRepoStub{workspace: &CcgoWorkspace{ID: 42, UserID: 7}}
	svc := NewCcgoService(repo, manager, runStarterStub{run: &CcgoWorkstationRun{WorkspaceID: 42, UserID: 7, RunID: "run_created", Status: CcgoRunStatusRunning}}, nil, nil)

	result, err := svc.StartWorkstation(context.Background(), CcgoStartWorkstationInput{UserID: 7, WorkspaceID: 42})
	require.NoError(t, err)
	require.NotNil(t, result.Agent)
	require.False(t, result.Reused)
	require.Equal(t, "run_created", result.Run.RunID)
	require.Equal(t, CcgoRunStatusRunning, result.Run.Status)
}

func TestCcgoServiceStartWorkstationReusesActiveRun(t *testing.T) {
	manager := hub.NewConnectionManager()
	manager.Register(42, noopTransport{})
	repo := &ccgoRepoStub{
		workspace: &CcgoWorkspace{ID: 42, UserID: 7},
	}
	svc := NewCcgoService(repo, manager, runStarterStub{
		run:    &CcgoWorkstationRun{WorkspaceID: 42, UserID: 7, RunID: "run_existing", Status: CcgoRunStatusRunning},
		reused: true,
	}, nil, nil)

	result, err := svc.StartWorkstation(context.Background(), CcgoStartWorkstationInput{UserID: 7, WorkspaceID: 42})
	require.NoError(t, err)
	require.True(t, result.Reused)
	require.Equal(t, "run_existing", result.Run.RunID)
}

func TestCcgoServiceStartWorkstationRejectsOtherUserWorkspace(t *testing.T) {
	manager := hub.NewConnectionManager()
	manager.Register(42, noopTransport{})
	svc := NewCcgoService(&ccgoRepoStub{workspace: &CcgoWorkspace{ID: 42, UserID: 99}}, manager, nil, nil, nil)

	_, err := svc.StartWorkstation(context.Background(), CcgoStartWorkstationInput{UserID: 7, WorkspaceID: 42})
	require.ErrorIs(t, err, ErrCcgoWorkspaceForbidden)
}

func TestCcgoServiceWorkstationStatusReportsDisconnectedResumableMapping(t *testing.T) {
	latest := &CcgoWorkstationRun{WorkspaceID: 42, UserID: 7, RunID: "run_old", Status: CcgoRunStatusStopped}
	svc := NewCcgoService(
		&ccgoRepoStub{workspace: &CcgoWorkspace{ID: 42, UserID: 7}, latestRun: latest},
		hub.NewConnectionManager(),
		runStarterStub{status: &CcgoRunnerStatus{WorkspaceID: 42}},
		nil,
		nil,
	)

	status, err := svc.WorkstationStatus(context.Background(), CcgoWorkstationStatusInput{UserID: 7, WorkspaceID: 42})
	require.NoError(t, err)
	require.False(t, status.AgentConnected)
	require.Equal(t, ErrCcgoAgentDisconnected.Message, status.AgentError)
	require.False(t, status.Runner.Running)
	require.True(t, status.Resumable)
	require.Equal(t, "run_old", status.LatestRun.RunID)
}

func TestCcgoServiceWorkstationStatusReportsRunningAgentAndRunner(t *testing.T) {
	manager := hub.NewConnectionManager()
	manager.Register(42, noopTransport{})
	svc := NewCcgoService(
		&ccgoRepoStub{workspace: &CcgoWorkspace{ID: 42, UserID: 7}, latestRun: &CcgoWorkstationRun{WorkspaceID: 42, UserID: 7, RunID: "run_live", Status: CcgoRunStatusRunning}},
		manager,
		runStarterStub{status: &CcgoRunnerStatus{WorkspaceID: 42, RunID: "run_live", Running: true, ProjectionMounted: true, ExecBridgeRunning: true, TerminalReady: true}},
		nil,
		nil,
	)

	status, err := svc.WorkstationStatus(context.Background(), CcgoWorkstationStatusInput{UserID: 7, WorkspaceID: 42})
	require.NoError(t, err)
	require.True(t, status.AgentConnected)
	require.Empty(t, status.AgentError)
	require.True(t, status.Runner.Running)
	require.False(t, status.Resumable)
}

func TestCcgoServiceStopWorkstationStopsRunnerAndAgent(t *testing.T) {
	manager := hub.NewConnectionManager()
	manager.Register(42, noopTransport{})
	stoppedRun := &CcgoWorkstationRun{WorkspaceID: 42, UserID: 7, RunID: "run_live", Status: CcgoRunStatusStopped}
	svc := NewCcgoService(
		&ccgoRepoStub{workspace: &CcgoWorkspace{ID: 42, UserID: 7}},
		manager,
		runStarterStub{
			status: &CcgoRunnerStatus{WorkspaceID: 42, RunID: "run_live", Running: false},
			stop:   stoppedRun,
		},
		nil,
		nil,
	)

	result, err := svc.StopWorkstation(context.Background(), CcgoStopWorkstationInput{UserID: 7, WorkspaceID: 42, Reason: "test stop"})
	require.NoError(t, err)
	require.True(t, result.Stopped)
	require.True(t, result.AgentDisconnected)
	require.Equal(t, CcgoRunStatusStopped, result.Run.Status)
	_, ok := manager.Get(42)
	require.False(t, ok)
}

func TestCcgoServiceStartDeviceLoginCreatesPendingRequest(t *testing.T) {
	repo := &ccgoRepoStub{}
	svc := NewCcgoService(repo, nil, nil, nil, nil)

	result, err := svc.StartDeviceLogin(context.Background(), "https://ccgo.test", "dev_123")
	require.NoError(t, err)
	require.NotEmpty(t, result.DeviceCode)
	require.NotEmpty(t, result.UserCode)
	require.Contains(t, result.VerificationURI, "https://ccgo.test/ccgo/device?code=")
	require.Equal(t, ccgoDeviceLoginPollIntervalSec, result.IntervalSeconds)
	require.Equal(t, HashCcgoSecret(result.DeviceCode), repo.deviceLogin.DeviceCodeHash)
	require.Equal(t, HashCcgoSecret(result.UserCode), repo.deviceLogin.UserCodeHash)
	require.Equal(t, "dev_123", repo.deviceLogin.DeviceID)
	require.Equal(t, CcgoDeviceLoginStatusPending, repo.deviceLogin.Status)
	require.Greater(t, repo.deviceLogin.ExpiresAt, time.Now())
}

func TestCcgoServicePollDeviceLoginPendingReturnsExplicitError(t *testing.T) {
	repo := &ccgoRepoStub{
		deviceLogin: &CcgoDeviceLogin{
			ID:             1,
			DeviceCodeHash: HashCcgoSecret("device_code"),
			Status:         CcgoDeviceLoginStatusPending,
			ExpiresAt:      time.Now().Add(time.Minute),
		},
	}
	svc := NewCcgoService(
		repo,
		nil,
		nil,
		deviceLoginUserReaderStub{},
		&deviceLoginTokenIssuerStub{},
	)

	_, err := svc.PollDeviceLogin(context.Background(), "device_code")
	require.ErrorIs(t, err, ErrCcgoDeviceLoginPending)
}

func TestCcgoServiceApproveAndPollDeviceLoginReturnsTokenAndConsumesRequest(t *testing.T) {
	repo := &ccgoRepoStub{}
	issuer := &deviceLoginTokenIssuerStub{
		pair:      &TokenPair{AccessToken: "access_123", RefreshToken: "refresh_123", ExpiresIn: 3600},
		expiresIn: 3600,
	}
	svc := NewCcgoService(
		repo,
		nil,
		nil,
		deviceLoginUserReaderStub{user: &User{ID: 7, Email: "alice@example.com", Role: RoleUser, Status: StatusActive}},
		issuer,
	)
	started, err := svc.StartDeviceLogin(context.Background(), "https://ccgo.test", "dev_123")
	require.NoError(t, err)

	approved, err := svc.ApproveDeviceLogin(context.Background(), 7, strings.ReplaceAll(strings.ToLower(started.UserCode), "-", ""))
	require.NoError(t, err)
	require.Equal(t, CcgoDeviceLoginStatusApproved, approved.Status)

	polled, err := svc.PollDeviceLogin(context.Background(), started.DeviceCode)
	require.NoError(t, err)
	require.Equal(t, "access_123", polled.AccessToken)
	require.Equal(t, "refresh_123", polled.RefreshToken)
	require.Equal(t, 3600, polled.ExpiresIn)
	require.Equal(t, "Bearer", polled.TokenType)
	require.Equal(t, CcgoDeviceLoginStatusConsumed, repo.deviceLogin.Status)
	require.Equal(t, []int64{7}, issuer.recordedIDs)
}

func TestCcgoServicePollDeviceLoginExpiresPendingRequest(t *testing.T) {
	repo := &ccgoRepoStub{
		deviceLogin: &CcgoDeviceLogin{
			ID:             1,
			DeviceCodeHash: HashCcgoSecret("device_code"),
			Status:         CcgoDeviceLoginStatusPending,
			ExpiresAt:      time.Now().Add(-time.Minute),
		},
	}
	svc := NewCcgoService(
		repo,
		nil,
		nil,
		deviceLoginUserReaderStub{},
		&deviceLoginTokenIssuerStub{},
	)

	_, err := svc.PollDeviceLogin(context.Background(), "device_code")
	require.ErrorIs(t, err, ErrCcgoDeviceLoginExpired)
	require.Equal(t, CcgoDeviceLoginStatusExpired, repo.deviceLogin.Status)
}
