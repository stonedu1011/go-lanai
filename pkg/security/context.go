package security

import (
	"context"
	tenant_hierarchy_accessor "cto-github.cisco.com/NFV-BU/go-lanai/pkg/tenant_hierarchy/accessor"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/utils"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"time"
)

const (
	HighestMiddlewareOrder = int(-1<<18 + 1)                 // -0x3ffff = -262143
	LowestMiddlewareOrder  = HighestMiddlewareOrder + 0xffff // -0x30000 = -196608
)

type AuthenticationState int

const (
	StateAnonymous = AuthenticationState(iota)
	StatePrincipalKnown
	StateAuthenticated
)

type Permissions map[string]interface{}

func (p Permissions) Has(permission string) bool {
	_, ok := p[permission]
	return ok
}

type Authentication interface {
	Principal() interface{}
	Permissions() Permissions
	State() AuthenticationState
	Details() interface{}
}

// EmptyAuthentication represent unauthenticated user.
// Note: anonymous user is considered authenticated
type EmptyAuthentication string

func (EmptyAuthentication) Principal() interface{} {
	return nil
}

func (EmptyAuthentication) Details() interface{} {
	return nil
}

func (EmptyAuthentication) State() AuthenticationState {
	return StateAnonymous
}

func (EmptyAuthentication) Permissions() Permissions {
	return map[string]interface{}{}
}

func GobRegister() {
	gob.Register(EmptyAuthentication(""))
	gob.Register((*AnonymousAuthentication)(nil))
	gob.Register((*CodedError)(nil))
	gob.Register(errors.New(""))
	gob.Register((*DefaultAccount)(nil))
	gob.Register((*AcctDetails)(nil))
	gob.Register((*AcctLockingRule)(nil))
	gob.Register((*AcctPasswordPolicy)(nil))
	gob.Register((*AccountMetadata)(nil))
}

func Get(ctx context.Context) Authentication {
	secCtx, ok := ctx.Value(ContextKeySecurity).(Authentication)
	if !ok {
		secCtx = EmptyAuthentication("EmptyAuthentication")
	}
	return secCtx
}

func Clear(ctx utils.MutableContext) {
	ctx.Set(gin.AuthUserKey, nil)
	ctx.Set(ContextKeySecurity, nil)
}

// TryClear attempt to clear security context. Return true if succeeded
func TryClear(ctx context.Context) bool {
	switch ctx.(type) {
	case utils.MutableContext:
		Clear(ctx.(utils.MutableContext))
	default:
		return false
	}
	return true
}

func HasPermissions(auth Authentication, permissions ...string) bool {
	for _, p := range permissions {
		if !auth.Permissions().Has(p) {
			return false
		}
	}
	return true
}

func HasAccessToTenant(ctx context.Context, auth Authentication, tenantId string) bool {
	if !tenant_hierarchy_accessor.IsLoaded(ctx) {
		logger.Warnf("Tenant hierarchy is not loaded by the auth server, hasAccessToTenant will not consider child tenants in the tenant hierarchy")
	}

	if ud, ok := auth.Details().(UserDetails); ok {
		if ud.AssignedTenantIds().Has(tenantId) {
			return true
		}
		ancestors, err := tenant_hierarchy_accessor.GetAnceostors(ctx, tenantId)
		if err != nil {
			return false
		}

		for _, ancestor := range ancestors {
			if ud.AssignedTenantIds().Has(ancestor) {
				return true
			}
		}
	}
	return false
}

func IsFullyAuthenticated(auth Authentication) bool {
	return auth.State() >= StateAuthenticated
}

func IsBeingAuthenticated(from, to Authentication) bool {
	fromUnauthenticatedState := from == nil || from.State() < StateAuthenticated
	toAuthenticatedState := to != nil && to.State() > StatePrincipalKnown
	return fromUnauthenticatedState && toAuthenticatedState
}

func IsBeingUnAuthenticated(from, to Authentication) bool {
	fromAuthenticated := from != nil && from.State() > StateAnonymous
	toUnAuthenticatedState := to == nil || to.State() <= StateAnonymous
	return fromAuthenticated && toUnAuthenticatedState
}

func DetermineAuthenticationTime(ctx context.Context, userAuth Authentication) (authTime time.Time) {
	if userAuth == nil {
		return
	}

	details, ok := userAuth.Details().(map[string]interface{})
	if !ok {
		return
	}

	v, ok := details[DetailsKeyAuthTime]
	if !ok {
		return
	}

	switch v.(type) {
	case time.Time:
		authTime = v.(time.Time)
	case string:
		authTime = utils.ParseTime(utils.ISO8601Milliseconds, v.(string))
	}
	return
}

func GetUsername(userAuth Authentication) (string, error){
	principal := userAuth.Principal()
	var username string
	switch principal.(type) {
	case Account:
		username = principal.(Account).Username()
	case string:
		username = principal.(string)
	case fmt.Stringer:
		username = principal.(fmt.Stringer).String()
	default:
		return "", errors.New(fmt.Sprintf("unsupported principal type %T", principal))
	}
	return username, nil
}