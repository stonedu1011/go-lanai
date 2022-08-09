package logout

import (
	"context"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"fmt"
	"net/http"
)

/*********************************
	Feature Impl
 *********************************/

//goland:noinspection GoNameStartsWithPackageName
type LogoutHandler interface {
	// HandleLogout is the method MW would use to perform logging out actions.
	// In case of multiple LogoutHandler are registered, implementing class can terminate logout by implementing ConditionalLogoutHandler
	HandleLogout(context.Context, *http.Request, http.ResponseWriter, security.Authentication) error
}

// ConditionalLogoutHandler is a supplementary interface for LogoutHandler.
// It's capable of cancelling/delaying logout process before any LogoutHandler is executed.
// When non-nil error is returned and logout middleware is configured with an security.AuthenticationEntryPoint,
// the entry point is used to delay the logout process
// In case of multiple ConditionalLogoutHandler, returning error by any handler would immediately terminate the process
type ConditionalLogoutHandler interface {
	// ShouldLogout returns error if logging out cannot be performed.
	ShouldLogout(context.Context, *http.Request, http.ResponseWriter, security.Authentication) error
}

//goland:noinspection GoNameStartsWithPackageName
type LogoutFeature struct {
	successHandler security.AuthenticationSuccessHandler
	errorHandler   security.AuthenticationErrorHandler
	entryPoint     security.AuthenticationEntryPoint
	successUrl     string
	errorUrl       string
	logoutHandlers []LogoutHandler
	logoutUrl      string
}

// Identifier Standard security.Feature entrypoint
func (f *LogoutFeature) Identifier() security.FeatureIdentifier {
	return FeatureId
}

// LogoutHandlers override default handler
func (f *LogoutFeature) LogoutHandlers(logoutHandlers ...LogoutHandler) *LogoutFeature {
	f.logoutHandlers = logoutHandlers
	return f
}

func (f *LogoutFeature) AddLogoutHandler(logoutHandler LogoutHandler) *LogoutFeature {
	f.logoutHandlers = append([]LogoutHandler{logoutHandler}, f.logoutHandlers...)
	return f
}

func (f *LogoutFeature) LogoutUrl(logoutUrl string) *LogoutFeature {
	f.logoutUrl = logoutUrl
	return f
}

func (f *LogoutFeature) SuccessUrl(successUrl string) *LogoutFeature {
	f.successUrl = successUrl
	return f
}

func (f *LogoutFeature) ErrorUrl(errorUrl string) *LogoutFeature {
	f.errorUrl = errorUrl
	return f
}

// SuccessHandler overrides SuccessUrl
func (f *LogoutFeature) SuccessHandler(successHandler security.AuthenticationSuccessHandler) *LogoutFeature {
	f.successHandler = successHandler
	return f
}

// ErrorHandler overrides ErrorUrl
func (f *LogoutFeature) ErrorHandler(errorHandler security.AuthenticationErrorHandler) *LogoutFeature {
	f.errorHandler = errorHandler
	return f
}

// EntryPoint is used when ConditionalLogoutHandler decide cancel/delay logout process
func (f *LogoutFeature) EntryPoint(entryPoint security.AuthenticationEntryPoint) *LogoutFeature {
	f.entryPoint = entryPoint
	return f
}

/*********************************
	Constructors and Configure
 *********************************/

// Configure security.Feature entrypoint, used for modifying existing configuration in given security.WebSecurity
func Configure(ws security.WebSecurity) *LogoutFeature {
	feature := New()
	if fc, ok := ws.(security.FeatureModifier); ok {
		return fc.Enable(feature).(*LogoutFeature)
	}
	panic(fmt.Errorf("unable to configure form login: provided WebSecurity [%T] doesn't support FeatureModifier", ws))
}

// New Standard security.Feature entrypoint, DSL style. Used with security.WebSecurity
func New() *LogoutFeature {
	return &LogoutFeature{
		successUrl: "/login",
		logoutUrl:  "/logout",
		logoutHandlers: []LogoutHandler{
			DefaultLogoutHandler{},
		},
	}
}

type DefaultLogoutHandler struct{}

func (h DefaultLogoutHandler) HandleLogout(ctx context.Context, _ *http.Request, _ http.ResponseWriter, _ security.Authentication) error {
	security.Clear(ctx)
	return nil
}
