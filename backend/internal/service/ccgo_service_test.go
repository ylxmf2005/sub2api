package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/hub"
	"github.com/Wei-Shaw/sub2api/internal/ccgo/protocol"
	"github.com/stretchr/testify/require"
)

type ccgoRepoStub struct {
	workspace *CcgoWorkspace
	activeRun *CcgoWorkstationRun
	created   *CcgoWorkstationRun
	running   *CcgoWorkstationRun
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

func (r *ccgoRepoStub) MarkWorkstationRunFailed(context.Context, string, string, time.Time) error {
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

func TestCcgoServiceStartWorkstationRequiresAgentConnection(t *testing.T) {
	svc := NewCcgoService(&ccgoRepoStub{workspace: &CcgoWorkspace{ID: 42, UserID: 7}}, hub.NewConnectionManager())

	_, err := svc.StartWorkstation(context.Background(), CcgoStartWorkstationInput{UserID: 7, WorkspaceID: 42})
	require.ErrorIs(t, err, ErrCcgoAgentDisconnected)
}

func TestCcgoServiceStartWorkstationCreatesRunWhenAgentConnected(t *testing.T) {
	manager := hub.NewConnectionManager()
	manager.Register(42, noopTransport{})
	repo := &ccgoRepoStub{workspace: &CcgoWorkspace{ID: 42, UserID: 7}}
	svc := NewCcgoService(repo, manager)

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
		activeRun: &CcgoWorkstationRun{WorkspaceID: 42, UserID: 7, RunID: "run_existing", Status: CcgoRunStatusRunning},
	}
	svc := NewCcgoService(repo, manager)

	result, err := svc.StartWorkstation(context.Background(), CcgoStartWorkstationInput{UserID: 7, WorkspaceID: 42})
	require.NoError(t, err)
	require.True(t, result.Reused)
	require.Equal(t, "run_existing", result.Run.RunID)
	require.Nil(t, repo.created)
}

func TestCcgoServiceStartWorkstationRejectsOtherUserWorkspace(t *testing.T) {
	manager := hub.NewConnectionManager()
	manager.Register(42, noopTransport{})
	svc := NewCcgoService(&ccgoRepoStub{workspace: &CcgoWorkspace{ID: 42, UserID: 99}}, manager)

	_, err := svc.StartWorkstation(context.Background(), CcgoStartWorkstationInput{UserID: 7, WorkspaceID: 42})
	require.ErrorIs(t, err, ErrCcgoWorkspaceForbidden)
}
