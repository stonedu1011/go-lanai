package passwd

import (
	"context"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/utils/order"
	"errors"
	"time"
)

/******************************
	abstracts
 ******************************/
// AuthenticationResult is a values carrier for PostAuthenticationProcessor
type AuthenticationResult struct {
	Candidate security.Candidate
	Auth      security.Authentication
	Error     error
}

// PostAuthenticationProcessor is invoked at the end of authentication process regardless of authentication decisions (granted or rejected)
// If PostAuthenticationProcessor implement order.Ordered interface, its order is respected using order.OrderedFirstCompareReverse.
// This means highest priority is executed last
type PostAuthenticationProcessor interface {
	// Process is invoked at the end of authentication process by the Authenticator to perform post-auth action.
	// The method is invoked regardless if the authentication is granted:
	// 	- If the authentication is granted, the AuthenticationResult.Auth is non-nil and AuthenticationResult.Error is nil
	//	- If the authentication is rejected, the AuthenticationResult.Error is non-nil and AuthenticationResult.Auth should be ignored
	//
	// If the context.Context and security.Account paramters are mutable, PostAuthenticationProcessor is allowed to change it
	// Note: PostAuthenticationProcessor typically shouldn't overwrite authentication decision (rejected or approved)
	// 		 However, it is allowed to modify result by returning different AuthenticationResult.
	//       This is useful when PostAuthenticationProcessor want to returns different error or add more details to authentication
	Process(context.Context, security.Account, AuthenticationResult) AuthenticationResult
}

/******************************
	Helpers
 ******************************/
func applyPostAuthenticationProcessors(processors []PostAuthenticationProcessor,
	ctx context.Context, acct security.Account, can security.Candidate, auth security.Authentication, err error) (security.Authentication, error) {

	result := AuthenticationResult{
		Candidate: can,
		Auth: auth,
		Error: err,
	}
	for _,processor := range processors {
		result = processor.Process(ctx, acct, result)
	}
	return result.Auth, result.Error
}

/******************************
	Common Implementation
 ******************************/
// PersistAccountPostProcessor saves Account. It's implement order.Ordered with highest priority
// Note: post-processors executed in reversed order
type PersistAccountPostProcessor struct {
	store security.AccountStore
}

func NewPersistAccountPostProcessor(store security.AccountStore) *PersistAccountPostProcessor {
	return &PersistAccountPostProcessor{store: store}
}

// run last
func (p *PersistAccountPostProcessor) Order() int {
	return order.Highest
}

func (p *PersistAccountPostProcessor) Process(ctx context.Context, acct security.Account, result AuthenticationResult) AuthenticationResult {
	if acct == nil {
		return result
	}

	// regardless decision, account need to be persisted in case of any status changes.
	// Note: we ignore save error since it's too late to do anything
	_ = p.store.Save(ctx, acct)
	return result
}

// AccountStatusPostProcessor updates account based on authentication result.
// It could update last login status, failed login status, etc.
type AccountStatusPostProcessor struct {
	store security.AccountStore
}

func NewAccountStatusPostProcessor(store security.AccountStore) *AccountStatusPostProcessor {
	return &AccountStatusPostProcessor{store: store}
}

// run first
func (p *AccountStatusPostProcessor) Order() int {
	return order.Lowest
}

func (p *AccountStatusPostProcessor) Process(ctx context.Context, acct security.Account, result AuthenticationResult) AuthenticationResult {
	updater, ok := acct.(security.AccountUpdater)
	if !ok {
		return result
	}

	switch {
	case result.Error == nil && result.Auth != nil && result.Auth.State() >= security.StateAuthenticated:
		// fully authenticated
		updater.RecordSuccess(time.Now())
		updater.ResetFailedAttempts()
	case errors.Is(result.Error, errorBadCredentials) && isPasswordAuth(result):
		// Password auth failed with incorrect password
		limit := 5
		if rules, e := p.store.LoadLockingRules(ctx, acct); e == nil && rules != nil && rules.LockoutEnabled() {
			limit = rules.LockoutFailuresLimit()
		}
		updater.RecordFailure(time.Now(), limit)
	default:
	}

	return result
}

// AccountLockingPostProcessor react on failed authentication. Lock account if necessary
type AccountLockingPostProcessor struct {
	store security.AccountStore
}

func NewAccountLockingPostProcessor(store security.AccountStore) *AccountLockingPostProcessor {
	return &AccountLockingPostProcessor{store: store}
}

// run between AccountStatusPostProcessor and AccountStatusPostProcessor
func (p *AccountLockingPostProcessor) Order() int {
	return 0
}

func (p *AccountLockingPostProcessor) Process(ctx context.Context, acct security.Account, result AuthenticationResult) AuthenticationResult {
	// skip if
	// 1. account is not updatable
	// 2. not bad credentials
	// 3. not password auth
	updater, ok := acct.(security.AccountUpdater)
	if !ok || !errors.Is(result.Error, errorBadCredentials) || !isPasswordAuth(result) {
		return result
	}

	history, ok := acct.(security.AccountHistory)
	rules, e := p.store.LoadLockingRules(ctx, acct)
	if !ok || e != nil || rules == nil || !rules.LockoutEnabled() {
		return result
	}

	// Note: we assume AccountStatusPostProcessor already updated login success/failure records
	// find first login failure within FailureInterval
	now := time.Now()
	var count int
	for _,t := range history.LoginFailures() {
		if interval := now.Sub(t); interval >= 0 && interval <= rules.LockoutFailuresInterval() {
			count ++
		}
	}

	// lock the account if over the limit
	if count >= rules.LockoutFailuresLimit() {
		updater.Lock()
		// Optional, change error message
		result.Error = security.NewAccountStatusError(MessageAccountLockedWithReason, result.Error)
	}
	return result
}

/******************************
	Helper
 ******************************/
func isPasswordAuth(result AuthenticationResult) bool {
	_, ok := result.Candidate.(*UsernamePasswordPair);
	return ok
}

func isMfaVerify(result AuthenticationResult) bool {
	_, ok := result.Candidate.(*MFAOtpVerification);
	return ok
}