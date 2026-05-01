package service

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

var (
	ErrResourceSupplySelfServiceDisabled = infraerrors.Forbidden("RESOURCE_SUPPLY_SELF_SERVICE_DISABLED", "resource supply self-service intake is disabled")
	ErrResourceSupplyGroupUnsupported    = infraerrors.BadRequest("RESOURCE_SUPPLY_GROUP_UNSUPPORTED", "group does not support resource supply intake")
	ErrResourceSupplyAccountForbidden    = infraerrors.Forbidden("RESOURCE_SUPPLY_ACCOUNT_FORBIDDEN", "resource supply account is not owned by the user")
	ErrResourceSupplyInvalidStatus       = infraerrors.BadRequest("RESOURCE_SUPPLY_INVALID_STATUS", "invalid resource supply status")
	ErrResourceSupplyBalanceEmpty        = infraerrors.BadRequest("RESOURCE_SUPPLY_BALANCE_EMPTY", "no resource supply earnings available to transfer")
	ErrResourceSupplyAdjustmentNote      = infraerrors.BadRequest("RESOURCE_SUPPLY_ADJUSTMENT_NOTE_REQUIRED", "resource supply adjustment note is required")
	ErrResourceSupplyWouldOverdraw       = infraerrors.BadRequest("RESOURCE_SUPPLY_WOULD_OVERDRAW", "resource supply adjustment would overdraw available earnings")
)

type ResourceSupplyBalance struct {
	UserID                    int64     `json:"user_id"`
	AvailableAmount           float64   `json:"available_amount"`
	LifetimeEarnedAmount      float64   `json:"lifetime_earned_amount"`
	LifetimeTransferredAmount float64   `json:"lifetime_transferred_amount"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

type ResourceSupplyOwnedAccount struct {
	ID                 int64      `json:"id"`
	Name               string     `json:"name"`
	Platform           string     `json:"platform"`
	Type               string     `json:"type"`
	GroupIDs           []int64    `json:"group_ids"`
	GroupNames         []string   `json:"group_names"`
	SupplySource       string     `json:"supply_source"`
	SupplyStatus       string     `json:"supply_status"`
	SupplyStatusReason *string    `json:"supply_status_reason,omitempty"`
	Schedulable        bool       `json:"schedulable"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	ReviewedAt         *time.Time `json:"reviewed_at,omitempty"`
}

type ResourceSupplyLedgerEntry struct {
	ID                  int64     `json:"id"`
	OwnerUserID         int64     `json:"owner_user_id"`
	CallerUserID        *int64    `json:"caller_user_id,omitempty"`
	APIKeyID            *int64    `json:"api_key_id,omitempty"`
	GroupID             *int64    `json:"group_id,omitempty"`
	AccountID           *int64    `json:"account_id,omitempty"`
	UsageBillingEventID *int64    `json:"usage_billing_event_id,omitempty"`
	LedgerType          string    `json:"ledger_type"`
	Amount              float64   `json:"amount"`
	BalanceAfter        float64   `json:"balance_after"`
	ActualCost          *float64  `json:"actual_cost,omitempty"`
	RewardMultiplier    *float64  `json:"reward_multiplier,omitempty"`
	BillingType         *int8     `json:"billing_type,omitempty"`
	Model               *string   `json:"model,omitempty"`
	RequestID           *string   `json:"request_id,omitempty"`
	AdminUserID         *int64    `json:"admin_user_id,omitempty"`
	Note                *string   `json:"note,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	OwnerEmail          *string   `json:"owner_email,omitempty"`
	CallerEmail         *string   `json:"caller_email,omitempty"`
	GroupName           *string   `json:"group_name,omitempty"`
	AccountName         *string   `json:"account_name,omitempty"`
}

type ResourceSupplySummary struct {
	Balance             ResourceSupplyBalance        `json:"balance"`
	AccountStatusCounts map[string]int               `json:"account_status_counts"`
	Accounts            []ResourceSupplyOwnedAccount `json:"accounts"`
	RecentLedger        []ResourceSupplyLedgerEntry  `json:"recent_ledger"`
	SelfServiceEnabled  bool                         `json:"self_service_enabled"`
}

type ResourceSupplyLedgerFilter struct {
	LedgerType string
	Page       int
	PageSize   int
}

type ResourceSupplyAdminLedgerFilter struct {
	OwnerUserID  *int64
	CallerUserID *int64
	GroupID      *int64
	AccountID    *int64
	LedgerType   string
	RequestID    string
	Page         int
	PageSize     int
}

type ResourceSupplyTransferResult struct {
	Amount     float64 `json:"amount"`
	NewBalance float64 `json:"new_balance"`
}

type ResourceSupplyAdjustmentInput struct {
	OwnerUserID int64
	AdminUserID int64
	Amount      float64
	Note        string
}

type ResourceSupplyRepository interface {
	GetUserSummary(ctx context.Context, userID int64, recentLimit int) (*ResourceSupplySummary, error)
	ListUserLedger(ctx context.Context, userID int64, filter ResourceSupplyLedgerFilter) ([]ResourceSupplyLedgerEntry, int64, error)
	ListAdminLedger(ctx context.Context, filter ResourceSupplyAdminLedgerFilter) ([]ResourceSupplyLedgerEntry, int64, error)
	TransferAvailableToBalance(ctx context.Context, userID int64, idempotencyKey string) (*ResourceSupplyTransferResult, error)
	AdjustBalance(ctx context.Context, input ResourceSupplyAdjustmentInput) (*ResourceSupplyLedgerEntry, error)
}

type ResourceSupplyService struct {
	repo                 ResourceSupplyRepository
	accountRepo          AccountRepository
	groupRepo            GroupRepository
	settingService       *SettingService
	accountTestService   *AccountTestService
	openAIOAuthService   *OpenAIOAuthService
	authCacheInvalidator APIKeyAuthCacheInvalidator
	billingCacheService  *BillingCacheService
}

func NewResourceSupplyService(
	repo ResourceSupplyRepository,
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	settingService *SettingService,
	accountTestService *AccountTestService,
	openAIOAuthService *OpenAIOAuthService,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	billingCacheService *BillingCacheService,
) *ResourceSupplyService {
	return &ResourceSupplyService{
		repo:                 repo,
		accountRepo:          accountRepo,
		groupRepo:            groupRepo,
		settingService:       settingService,
		accountTestService:   accountTestService,
		openAIOAuthService:   openAIOAuthService,
		authCacheInvalidator: authCacheInvalidator,
		billingCacheService:  billingCacheService,
	}
}

func (s *ResourceSupplyService) IsSelfServiceEnabled(ctx context.Context) bool {
	if s == nil || s.settingService == nil {
		return ResourceSupplySelfServiceEnabledDefault
	}
	return s.settingService.IsResourceSupplySelfServiceEnabled(ctx)
}

func (s *ResourceSupplyService) GetUserSummary(ctx context.Context, userID int64) (*ResourceSupplySummary, error) {
	if err := validateResourceSupplyUserID(userID); err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("RESOURCE_SUPPLY_UNAVAILABLE", "resource supply service unavailable")
	}
	summary, err := s.repo.GetUserSummary(ctx, userID, 20)
	if err != nil {
		return nil, err
	}
	summary.SelfServiceEnabled = s.IsSelfServiceEnabled(ctx)
	summary.RecentLedger = sanitizeUserResourceSupplyLedgerEntries(summary.RecentLedger)
	return summary, nil
}

func (s *ResourceSupplyService) ListUserLedger(ctx context.Context, userID int64, filter ResourceSupplyLedgerFilter) ([]ResourceSupplyLedgerEntry, int64, error) {
	if err := validateResourceSupplyUserID(userID); err != nil {
		return nil, 0, err
	}
	if s == nil || s.repo == nil {
		return nil, 0, infraerrors.ServiceUnavailable("RESOURCE_SUPPLY_UNAVAILABLE", "resource supply service unavailable")
	}
	entries, total, err := s.repo.ListUserLedger(ctx, userID, normalizeResourceSupplyLedgerFilter(filter))
	if err != nil {
		return nil, 0, err
	}
	return sanitizeUserResourceSupplyLedgerEntries(entries), total, nil
}

func (s *ResourceSupplyService) ListAdminLedger(ctx context.Context, filter ResourceSupplyAdminLedgerFilter) ([]ResourceSupplyLedgerEntry, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, infraerrors.ServiceUnavailable("RESOURCE_SUPPLY_UNAVAILABLE", "resource supply service unavailable")
	}
	filter.Page, filter.PageSize = normalizeResourceSupplyPage(filter.Page, filter.PageSize)
	filter.LedgerType = normalizeResourceSupplyLedgerType(filter.LedgerType)
	return s.repo.ListAdminLedger(ctx, filter)
}

func (s *ResourceSupplyService) TransferAvailableToBalance(ctx context.Context, userID int64, idempotencyKey string) (*ResourceSupplyTransferResult, error) {
	if err := validateResourceSupplyUserID(userID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, infraerrors.BadRequest("IDEMPOTENCY_KEY_REQUIRED", "idempotency key is required")
	}
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("RESOURCE_SUPPLY_UNAVAILABLE", "resource supply service unavailable")
	}
	result, err := s.repo.TransferAvailableToBalance(ctx, userID, strings.TrimSpace(idempotencyKey))
	if err != nil {
		return nil, err
	}
	s.invalidateUserBalance(ctx, userID)
	return result, nil
}

func (s *ResourceSupplyService) AdjustBalance(ctx context.Context, input ResourceSupplyAdjustmentInput) (*ResourceSupplyLedgerEntry, error) {
	if err := validateResourceSupplyUserID(input.OwnerUserID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Note) == "" {
		return nil, ErrResourceSupplyAdjustmentNote
	}
	if input.Amount == 0 || math.IsNaN(input.Amount) || math.IsInf(input.Amount, 0) {
		return nil, infraerrors.BadRequest("RESOURCE_SUPPLY_INVALID_AMOUNT", "resource supply adjustment amount is invalid")
	}
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("RESOURCE_SUPPLY_UNAVAILABLE", "resource supply service unavailable")
	}
	entry, err := s.repo.AdjustBalance(ctx, input)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

type ResourceSupplyOpenAIAPIKeyInput struct {
	UserID  int64
	GroupID int64
	Name    string
	APIKey  string
	BaseURL string
	ModelID string
}

func (s *ResourceSupplyService) SubmitOpenAIAPIKey(ctx context.Context, input ResourceSupplyOpenAIAPIKeyInput) (*Account, error) {
	if err := validateResourceSupplyUserID(input.UserID); err != nil {
		return nil, err
	}
	apiKey := strings.TrimSpace(input.APIKey)
	if apiKey == "" {
		return nil, infraerrors.BadRequest("OPENAI_API_KEY_REQUIRED", "openai api key is required")
	}
	group, err := s.validateSelfServiceGroup(ctx, input.GroupID, AccountTypeAPIKey)
	if err != nil {
		return nil, err
	}
	if group.RequireOAuthOnly {
		return nil, infraerrors.BadRequest("RESOURCE_SUPPLY_OAUTH_REQUIRED", "group requires OAuth accounts")
	}

	credentials := map[string]any{"api_key": apiKey}
	if baseURL := strings.TrimSpace(input.BaseURL); baseURL != "" {
		normalizedBaseURL, err := normalizeSelfServiceOpenAIBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		credentials["base_url"] = normalizedBaseURL
	}
	return s.createAndTestSelfServiceAccount(ctx, input.UserID, group, AccountTypeAPIKey, input.Name, credentials, nil, input.ModelID)
}

type ResourceSupplyOpenAIGenerateAuthURLInput struct {
	UserID      int64
	RedirectURI string
	ProxyID     *int64
}

func (s *ResourceSupplyService) GenerateOpenAIAuthURL(ctx context.Context, input ResourceSupplyOpenAIGenerateAuthURLInput) (*OpenAIAuthURLResult, error) {
	if err := validateResourceSupplyUserID(input.UserID); err != nil {
		return nil, err
	}
	if !s.IsSelfServiceEnabled(ctx) {
		return nil, ErrResourceSupplySelfServiceDisabled
	}
	if s == nil || s.openAIOAuthService == nil {
		return nil, infraerrors.ServiceUnavailable("OPENAI_OAUTH_UNAVAILABLE", "openai oauth service unavailable")
	}
	return s.openAIOAuthService.GenerateAuthURL(ctx, input.ProxyID, input.RedirectURI, PlatformOpenAI)
}

type ResourceSupplyOpenAIExchangeCodeInput struct {
	UserID      int64
	GroupID     int64
	Name        string
	SessionID   string
	Code        string
	State       string
	RedirectURI string
	ProxyID     *int64
	ModelID     string
}

func (s *ResourceSupplyService) ExchangeOpenAICode(ctx context.Context, input ResourceSupplyOpenAIExchangeCodeInput) (*Account, error) {
	if err := validateResourceSupplyUserID(input.UserID); err != nil {
		return nil, err
	}
	group, err := s.validateSelfServiceGroup(ctx, input.GroupID, AccountTypeOAuth)
	if err != nil {
		return nil, err
	}
	if s == nil || s.openAIOAuthService == nil {
		return nil, infraerrors.ServiceUnavailable("OPENAI_OAUTH_UNAVAILABLE", "openai oauth service unavailable")
	}
	tokenInfo, err := s.openAIOAuthService.ExchangeCode(ctx, &OpenAIExchangeCodeInput{
		SessionID:   input.SessionID,
		Code:        input.Code,
		State:       input.State,
		RedirectURI: input.RedirectURI,
		ProxyID:     input.ProxyID,
	})
	if err != nil {
		return nil, err
	}
	credentials := s.openAIOAuthService.BuildAccountCredentials(tokenInfo)
	extra := map[string]any{}
	if strings.TrimSpace(tokenInfo.PrivacyMode) != "" {
		extra["privacy_mode"] = strings.TrimSpace(tokenInfo.PrivacyMode)
	}
	return s.createAndTestSelfServiceAccount(ctx, input.UserID, group, AccountTypeOAuth, input.Name, credentials, extra, input.ModelID)
}

func (s *ResourceSupplyService) OwnerPauseAccount(ctx context.Context, userID, accountID int64, reason string) (*Account, error) {
	return s.updateOwnerAccountStatus(ctx, userID, accountID, ResourceSupplyStatusPaused, reason)
}

func (s *ResourceSupplyService) OwnerRevokeAccount(ctx context.Context, userID, accountID int64, reason string) (*Account, error) {
	return s.updateOwnerAccountStatus(ctx, userID, accountID, ResourceSupplyStatusRevoked, reason)
}

func (s *ResourceSupplyService) AdminApproveAccount(ctx context.Context, adminUserID, accountID int64, reason string) (*Account, error) {
	return s.updateAdminAccountStatus(ctx, adminUserID, accountID, ResourceSupplyStatusSchedulable, reason)
}

func (s *ResourceSupplyService) AdminRejectAccount(ctx context.Context, adminUserID, accountID int64, reason string) (*Account, error) {
	return s.updateAdminAccountStatus(ctx, adminUserID, accountID, ResourceSupplyStatusRejected, reason)
}

func (s *ResourceSupplyService) AdminPauseAccount(ctx context.Context, adminUserID, accountID int64, reason string) (*Account, error) {
	return s.updateAdminAccountStatus(ctx, adminUserID, accountID, ResourceSupplyStatusPaused, reason)
}

func (s *ResourceSupplyService) AdminResumeAccount(ctx context.Context, adminUserID, accountID int64, reason string) (*Account, error) {
	return s.updateAdminAccountStatus(ctx, adminUserID, accountID, ResourceSupplyStatusSchedulable, reason)
}

func (s *ResourceSupplyService) BindAccountOwner(ctx context.Context, adminUserID, accountID int64, ownerUserID *int64, status, reason string) (*Account, error) {
	if s == nil || s.accountRepo == nil {
		return nil, infraerrors.ServiceUnavailable("RESOURCE_SUPPLY_UNAVAILABLE", "resource supply service unavailable")
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if ownerUserID == nil || *ownerUserID <= 0 {
		account.SupplyOwnerUserID = nil
		account.SupplySource = nil
		account.SupplyStatus = ResourceSupplyStatusNone
		account.SupplyStatusReason = nil
		account.SupplySubmittedBy = nil
		account.SupplyReviewedBy = nil
		account.SupplyReviewedAt = nil
	} else {
		normalizedStatus := normalizeResourceSupplyStatusForWrite(status)
		if normalizedStatus == ResourceSupplyStatusNone {
			normalizedStatus = ResourceSupplyStatusSchedulable
		}
		if !IsValidResourceSupplyStatus(normalizedStatus) {
			return nil, ErrResourceSupplyInvalidStatus
		}
		source := ResourceSupplySourceAdmin
		account.SupplyOwnerUserID = ownerUserID
		account.SupplySource = &source
		account.SupplyStatus = normalizedStatus
		account.SupplyStatusReason = stringPointerOrNil(reason)
		account.SupplyReviewedBy = &adminUserID
		account.SupplyReviewedAt = &now
		account.Schedulable = normalizedStatus == ResourceSupplyStatusSchedulable
	}
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, err
	}
	return s.accountRepo.GetByID(ctx, accountID)
}

func (s *ResourceSupplyService) validateSelfServiceGroup(ctx context.Context, groupID int64, accountType string) (*Group, error) {
	if !s.IsSelfServiceEnabled(ctx) {
		return nil, ErrResourceSupplySelfServiceDisabled
	}
	if s == nil || s.groupRepo == nil {
		return nil, infraerrors.ServiceUnavailable("RESOURCE_SUPPLY_UNAVAILABLE", "resource supply service unavailable")
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil || !group.IsActive() || group.Platform != PlatformOpenAI || !group.SupplyRewardsEnabled {
		return nil, ErrResourceSupplyGroupUnsupported
	}
	if accountType == AccountTypeAPIKey && group.RequireOAuthOnly {
		return nil, infraerrors.BadRequest("RESOURCE_SUPPLY_OAUTH_REQUIRED", "group requires OAuth accounts")
	}
	return group, nil
}

func (s *ResourceSupplyService) createAndTestSelfServiceAccount(ctx context.Context, userID int64, group *Group, accountType, name string, credentials map[string]any, extra map[string]any, modelID string) (*Account, error) {
	if s == nil || s.accountRepo == nil || s.accountTestService == nil {
		return nil, infraerrors.ServiceUnavailable("RESOURCE_SUPPLY_TEST_UNAVAILABLE", "resource supply account testing is unavailable")
	}
	if group == nil {
		return nil, ErrResourceSupplyGroupUnsupported
	}
	source := ResourceSupplySourceSelfService
	submittedBy := userID
	statusReason := "connection test pending"
	account := &Account{
		Name:               resourceSupplyAccountName(name, accountType),
		Platform:           PlatformOpenAI,
		Type:               accountType,
		Credentials:        credentials,
		Extra:              normalizeResourceSupplyExtra(extra),
		Concurrency:        1,
		Priority:           50,
		Status:             StatusActive,
		Schedulable:        false,
		AutoPauseOnExpired: true,
		SupplyOwnerUserID:  &userID,
		SupplySource:       &source,
		SupplyStatus:       ResourceSupplyStatusTesting,
		SupplyStatusReason: &statusReason,
		SupplySubmittedBy:  &submittedBy,
		GroupIDs:           []int64{group.ID},
	}
	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, err
	}
	if err := s.accountRepo.BindGroups(ctx, account.ID, []int64{group.ID}); err != nil {
		if cleanupErr := s.accountRepo.Delete(ctx, account.ID); cleanupErr != nil {
			return nil, fmt.Errorf("bind resource supply account to group: %w (cleanup failed: %v)", err, cleanupErr)
		}
		return nil, fmt.Errorf("bind resource supply account to group: %w", err)
	}

	testResult, err := s.accountTestService.RunTestBackground(ctx, account.ID, strings.TrimSpace(modelID))
	if err != nil || testResult == nil || testResult.Status != "success" {
		reason := "connection test failed"
		if testResult != nil && strings.TrimSpace(testResult.ErrorMessage) != "" {
			reason = strings.TrimSpace(testResult.ErrorMessage)
		} else if err != nil {
			reason = err.Error()
		}
		account.SupplyStatus = ResourceSupplyStatusRejected
		account.SupplyStatusReason = &reason
		account.Schedulable = false
		if updateErr := s.accountRepo.Update(ctx, account); updateErr != nil {
			return nil, fmt.Errorf("mark resource supply account rejected after connection test failure: %w", updateErr)
		}
		return nil, infraerrors.New(http.StatusBadRequest, "RESOURCE_SUPPLY_CONNECTION_TEST_FAILED", reason)
	}

	now := time.Now()
	nextStatus := ResourceSupplyStatusPendingReview
	if group.SupplySelfServiceReviewPolicy == ResourceSupplyReviewPolicyAutoOnline {
		nextStatus = ResourceSupplyStatusSchedulable
		account.SupplyReviewedAt = &now
	}
	account.SupplyStatus = nextStatus
	account.Schedulable = nextStatus == ResourceSupplyStatusSchedulable
	account.SupplyStatusReason = nil
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, err
	}
	return s.accountRepo.GetByID(ctx, account.ID)
}

func (s *ResourceSupplyService) updateOwnerAccountStatus(ctx context.Context, userID, accountID int64, status, reason string) (*Account, error) {
	if err := validateResourceSupplyUserID(userID); err != nil {
		return nil, err
	}
	if s == nil || s.accountRepo == nil {
		return nil, infraerrors.ServiceUnavailable("RESOURCE_SUPPLY_UNAVAILABLE", "resource supply service unavailable")
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.SupplyOwnerUserID == nil || *account.SupplyOwnerUserID != userID || account.SupplySource == nil || *account.SupplySource != ResourceSupplySourceSelfService {
		return nil, ErrResourceSupplyAccountForbidden
	}
	account.SupplyStatus = status
	account.SupplyStatusReason = stringPointerOrNil(reason)
	account.Schedulable = false
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, err
	}
	return s.accountRepo.GetByID(ctx, accountID)
}

func (s *ResourceSupplyService) updateAdminAccountStatus(ctx context.Context, adminUserID, accountID int64, status, reason string) (*Account, error) {
	if s == nil || s.accountRepo == nil {
		return nil, infraerrors.ServiceUnavailable("RESOURCE_SUPPLY_UNAVAILABLE", "resource supply service unavailable")
	}
	if !IsValidResourceSupplyStatus(status) || status == ResourceSupplyStatusNone || status == ResourceSupplyStatusTesting {
		return nil, ErrResourceSupplyInvalidStatus
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.SupplyOwnerUserID == nil || *account.SupplyOwnerUserID <= 0 {
		return nil, ErrResourceSupplyAccountForbidden
	}
	now := time.Now()
	account.SupplyStatus = status
	account.SupplyStatusReason = stringPointerOrNil(reason)
	account.SupplyReviewedBy = &adminUserID
	account.SupplyReviewedAt = &now
	account.Schedulable = status == ResourceSupplyStatusSchedulable
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, err
	}
	return s.accountRepo.GetByID(ctx, accountID)
}

func (s *ResourceSupplyService) invalidateUserBalance(ctx context.Context, userID int64) {
	if s == nil || userID <= 0 {
		return
	}
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateUserBalance(ctx, userID)
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
}

func IsValidResourceSupplyStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case ResourceSupplyStatusNone,
		ResourceSupplyStatusTesting,
		ResourceSupplyStatusPendingReview,
		ResourceSupplyStatusSchedulable,
		ResourceSupplyStatusPaused,
		ResourceSupplyStatusRejected,
		ResourceSupplyStatusRevoked:
		return true
	default:
		return false
	}
}

func IsValidResourceSupplyReviewPolicy(policy string) bool {
	switch strings.TrimSpace(policy) {
	case ResourceSupplyReviewPolicyManualReview, ResourceSupplyReviewPolicyAutoOnline:
		return true
	default:
		return false
	}
}

func normalizeResourceSupplyReviewPolicyForWrite(policy string) string {
	policy = strings.TrimSpace(policy)
	if policy == "" {
		return ResourceSupplyReviewPolicyManualReview
	}
	return policy
}

func normalizeResourceSupplyStatusForWrite(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return ResourceSupplyStatusNone
	}
	return status
}

func normalizeResourceSupplyLedgerFilter(filter ResourceSupplyLedgerFilter) ResourceSupplyLedgerFilter {
	filter.Page, filter.PageSize = normalizeResourceSupplyPage(filter.Page, filter.PageSize)
	filter.LedgerType = normalizeResourceSupplyLedgerType(filter.LedgerType)
	return filter
}

func normalizeResourceSupplyPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func normalizeResourceSupplyLedgerType(ledgerType string) string {
	ledgerType = strings.TrimSpace(ledgerType)
	switch ledgerType {
	case "", ResourceSupplyLedgerTypeReward, ResourceSupplyLedgerTypeTransfer, ResourceSupplyLedgerTypeAdjustment:
		return ledgerType
	default:
		return ""
	}
}

func validateResourceSupplyUserID(userID int64) error {
	if userID <= 0 {
		return ErrUserNotFound
	}
	return nil
}

func resourceSupplyAccountName(name, accountType string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return fmt.Sprintf("OpenAI %s supply", strings.ReplaceAll(accountType, "_", " "))
}

func normalizeResourceSupplyExtra(extra map[string]any) map[string]any {
	if extra == nil {
		return map[string]any{}
	}
	return extra
}

func normalizeSelfServiceOpenAIBaseURL(raw string) (string, error) {
	normalized, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts:     []string{"api.openai.com"},
		RequireAllowlist: true,
		AllowPrivate:     false,
	})
	if err != nil {
		return "", infraerrors.BadRequest("RESOURCE_SUPPLY_BASE_URL_NOT_ALLOWED", "resource supply self-service base_url must be an official OpenAI HTTPS endpoint")
	}
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", infraerrors.BadRequest("RESOURCE_SUPPLY_BASE_URL_NOT_ALLOWED", "resource supply self-service base_url must be an official OpenAI HTTPS endpoint")
	}
	switch strings.TrimRight(parsed.Path, "/") {
	case "", "/v1", "/v1/responses":
	default:
		return "", infraerrors.BadRequest("RESOURCE_SUPPLY_BASE_URL_NOT_ALLOWED", "resource supply self-service base_url must be an official OpenAI HTTPS endpoint")
	}
	return normalized, nil
}

func sanitizeUserResourceSupplyLedgerEntries(entries []ResourceSupplyLedgerEntry) []ResourceSupplyLedgerEntry {
	if len(entries) == 0 {
		return entries
	}
	out := make([]ResourceSupplyLedgerEntry, len(entries))
	for i, entry := range entries {
		entry.CallerUserID = nil
		entry.APIKeyID = nil
		entry.UsageBillingEventID = nil
		entry.RequestID = nil
		entry.AdminUserID = nil
		entry.OwnerEmail = nil
		entry.CallerEmail = nil
		out[i] = entry
	}
	return out
}

func stringPointerOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
