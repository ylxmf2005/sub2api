//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type adminResourceSupplyAccountRepoStub struct {
	accountRepoStub
	account     *Account
	created     *Account
	createCalls int
	updateCalls int
}

func (r *adminResourceSupplyAccountRepoStub) Create(_ context.Context, account *Account) error {
	r.createCalls++
	if account.ID == 0 {
		account.ID = 101
	}
	r.created = account
	r.account = account
	return nil
}

func (r *adminResourceSupplyAccountRepoStub) GetByID(_ context.Context, _ int64) (*Account, error) {
	if r.account == nil {
		return nil, ErrAccountNotFound
	}
	return r.account, nil
}

func (r *adminResourceSupplyAccountRepoStub) Update(_ context.Context, account *Account) error {
	r.updateCalls++
	r.account = account
	return nil
}

func TestAdminService_CreateAccount_RejectsMissingSupplyOwnerUser(t *testing.T) {
	ownerID := int64(319126768)
	accountRepo := &adminResourceSupplyAccountRepoStub{}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		userRepo:    &userRepoStub{},
	}

	_, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "resource account",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeAPIKey,
		Credentials:          map[string]any{"api_key": "sk-test"},
		Concurrency:          1,
		Priority:             1,
		SkipDefaultGroupBind: true,
		SupplyOwnerUserID:    &ownerID,
	})

	require.ErrorIs(t, err, ErrSupplyOwnerUserNotFound)
	require.Zero(t, accountRepo.createCalls)
}

func TestAdminService_UpdateAccount_RejectsMissingSupplyOwnerUserBeforePersist(t *testing.T) {
	ownerID := int64(319126768)
	accountRepo := &adminResourceSupplyAccountRepoStub{
		account: &Account{
			ID:       24,
			Name:     "qhrPlus",
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		userRepo:    &userRepoStub{},
	}

	_, err := svc.UpdateAccount(context.Background(), 24, &UpdateAccountInput{
		SupplyOwnerUserID: &ownerID,
	})

	require.ErrorIs(t, err, ErrSupplyOwnerUserNotFound)
	require.Zero(t, accountRepo.updateCalls)
}

func TestAdminService_UpdateAccount_BindsExistingSupplyOwnerUser(t *testing.T) {
	ownerID := int64(2)
	accountRepo := &adminResourceSupplyAccountRepoStub{
		account: &Account{
			ID:       24,
			Name:     "qhrPlus",
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		userRepo:    &userRepoStub{user: &User{ID: ownerID, Email: "319126768@qq.com"}},
	}

	updated, err := svc.UpdateAccount(context.Background(), 24, &UpdateAccountInput{
		SupplyOwnerUserID: &ownerID,
	})

	require.NoError(t, err)
	require.Equal(t, 1, accountRepo.updateCalls)
	require.NotNil(t, updated.SupplyOwnerUserID)
	require.Equal(t, ownerID, *updated.SupplyOwnerUserID)
	require.NotNil(t, updated.SupplySource)
	require.Equal(t, ResourceSupplySourceAdmin, *updated.SupplySource)
	require.Equal(t, ResourceSupplyStatusSchedulable, updated.SupplyStatus)
}
