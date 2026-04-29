//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type apiKeySettlementGroupRepoStub struct {
	groupRepoNoop
	activeGroups []Group
}

func (s *apiKeySettlementGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	return s.activeGroups, nil
}

type apiKeySettlementUserSubRepoStub struct {
	userSubRepoNoop
	activeSubs []UserSubscription
}

func (s *apiKeySettlementUserSubRepoStub) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	return append([]UserSubscription(nil), s.activeSubs...), nil
}

type apiKeySettlementReaderStub struct {
	groupIDs []int64
}

func (s *apiKeySettlementReaderStub) IsCurrentParticipant(context.Context, int64, int64) (bool, error) {
	return false, nil
}

func (s *apiKeySettlementReaderStub) ListCurrentParticipantGroupIDs(context.Context, int64) ([]int64, error) {
	return append([]int64(nil), s.groupIDs...), nil
}

func TestAPIKeyService_GetAvailableGroups_IncludesCurrentSettlementPoolParticipants(t *testing.T) {
	const (
		userID              int64 = 42
		settlementGroupID   int64 = 1001
		standardGroupID     int64 = 1002
		subscriptionGroupID int64 = 1003
	)

	userRepo := &mockUserRepo{
		getByIDUser: &User{
			ID:            userID,
			Status:        StatusActive,
			AllowedGroups: nil,
		},
	}
	groupRepo := &apiKeySettlementGroupRepoStub{
		activeGroups: []Group{
			{ID: settlementGroupID, Name: "settlement", Status: StatusActive, SubscriptionType: SubscriptionTypeSettlementPool},
			{ID: standardGroupID, Name: "public", Status: StatusActive, SubscriptionType: SubscriptionTypeStandard},
			{ID: subscriptionGroupID, Name: "sub", Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription},
		},
	}
	userSubRepo := &apiKeySettlementUserSubRepoStub{
		activeSubs: []UserSubscription{
			{UserID: userID, GroupID: subscriptionGroupID, Status: SubscriptionStatusActive},
		},
	}
	settlementReader := &apiKeySettlementReaderStub{
		groupIDs: []int64{settlementGroupID},
	}

	svc := NewAPIKeyService(
		&quotaBaseAPIKeyRepoStub{},
		userRepo,
		groupRepo,
		userSubRepo,
		nil,
		nil,
		&config.Config{},
		settlementReader,
	)

	groups, err := svc.GetAvailableGroups(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, groups, 3)
	require.Equal(t, []int64{settlementGroupID, standardGroupID, subscriptionGroupID}, []int64{groups[0].ID, groups[1].ID, groups[2].ID})
}
