package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeUserResourceSupplyLedgerEntries_RemovesInternalAuditFields(t *testing.T) {
	t.Parallel()

	callerID := int64(11)
	apiKeyID := int64(12)
	eventID := int64(13)
	adminID := int64(14)
	ownerEmail := "owner@example.com"
	callerEmail := "caller@example.com"
	requestID := "req-sensitive"
	accountID := int64(21)
	groupID := int64(22)

	entries := sanitizeUserResourceSupplyLedgerEntries([]ResourceSupplyLedgerEntry{
		{
			OwnerUserID:         10,
			CallerUserID:        &callerID,
			APIKeyID:            &apiKeyID,
			GroupID:             &groupID,
			AccountID:           &accountID,
			UsageBillingEventID: &eventID,
			RequestID:           &requestID,
			AdminUserID:         &adminID,
			OwnerEmail:          &ownerEmail,
			CallerEmail:         &callerEmail,
			GroupName:           stringPointerOrNil("supply-group"),
			AccountName:         stringPointerOrNil("owner-account"),
		},
	})

	require.Len(t, entries, 1)
	entry := entries[0]
	require.Nil(t, entry.CallerUserID)
	require.Nil(t, entry.APIKeyID)
	require.Nil(t, entry.UsageBillingEventID)
	require.Nil(t, entry.RequestID)
	require.Nil(t, entry.AdminUserID)
	require.Nil(t, entry.OwnerEmail)
	require.Nil(t, entry.CallerEmail)
	require.Equal(t, accountID, *entry.AccountID)
	require.Equal(t, groupID, *entry.GroupID)
	require.Equal(t, "supply-group", *entry.GroupName)
	require.Equal(t, "owner-account", *entry.AccountName)
}

func TestNormalizeSelfServiceOpenAIBaseURL_AllowsOfficialOpenAIOnly(t *testing.T) {
	t.Parallel()

	normalized, err := normalizeSelfServiceOpenAIBaseURL("https://api.openai.com/v1/")
	require.NoError(t, err)
	require.Equal(t, "https://api.openai.com/v1", normalized)

	_, err = normalizeSelfServiceOpenAIBaseURL("http://api.openai.com/v1")
	require.Error(t, err)

	_, err = normalizeSelfServiceOpenAIBaseURL("https://127.0.0.1/v1")
	require.Error(t, err)

	_, err = normalizeSelfServiceOpenAIBaseURL("https://openai-compatible.example/v1")
	require.Error(t, err)

	_, err = normalizeSelfServiceOpenAIBaseURL("https://api.openai.com/v1?next=https://example.com")
	require.Error(t, err)

	_, err = normalizeSelfServiceOpenAIBaseURL("https://user:pass@api.openai.com/v1")
	require.Error(t, err)
}

func TestBuildUsageBillingCommand_IncludesResourceSupplyRewardSnapshot(t *testing.T) {
	t.Parallel()

	groupID := int64(31)
	ownerID := int64(32)

	cmd := buildUsageBillingCommand("req-supply", nil, &postUsageBillingParams{
		Cost: &CostBreakdown{TotalCost: 1.5, ActualCost: 2.25},
		User: &User{ID: 40},
		APIKey: &APIKey{
			ID:      41,
			GroupID: &groupID,
			Group: &Group{
				ID:                     groupID,
				SupplyRewardsEnabled:   true,
				SupplyRewardMultiplier: 1.75,
			},
		},
		Account: &Account{
			ID:                42,
			Type:              AccountTypeAPIKey,
			SupplyOwnerUserID: &ownerID,
			SupplyStatus:      ResourceSupplyStatusSchedulable,
		},
	})

	require.NotNil(t, cmd)
	require.True(t, cmd.SupplyRewardEligible)
	require.Equal(t, ownerID, *cmd.SupplyOwnerUserID)
	require.Equal(t, 1.75, cmd.SupplyRewardMultiplier)
	require.Equal(t, ResourceSupplyStatusSchedulable, cmd.SupplyAccountStatus)
}

func TestBuildUsageBillingCommand_DoesNotRewardOwnerSelfUse(t *testing.T) {
	t.Parallel()

	groupID := int64(31)
	ownerID := int64(32)

	cmd := buildUsageBillingCommand("req-supply-self", nil, &postUsageBillingParams{
		Cost: &CostBreakdown{TotalCost: 1.5, ActualCost: 2.25},
		User: &User{ID: ownerID},
		APIKey: &APIKey{
			ID:      41,
			GroupID: &groupID,
			Group: &Group{
				ID:                     groupID,
				SupplyRewardsEnabled:   true,
				SupplyRewardMultiplier: 1.75,
			},
		},
		Account: &Account{
			ID:                42,
			Type:              AccountTypeAPIKey,
			SupplyOwnerUserID: &ownerID,
			SupplyStatus:      ResourceSupplyStatusSchedulable,
		},
	})

	require.NotNil(t, cmd)
	require.False(t, cmd.SupplyRewardEligible)
	require.Equal(t, ownerID, *cmd.SupplyOwnerUserID)
	require.Equal(t, 1.75, cmd.SupplyRewardMultiplier)
}
