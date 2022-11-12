package sectest

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/session"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/web"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/web/matcher"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/web/middleware"
	"cto-github.cisco.com/NFV-BU/go-lanai/test"
	"cto-github.cisco.com/NFV-BU/go-lanai/test/apptest"
	"fmt"
	"go.uber.org/fx"
	"net/http"
)

/**************************
	Context
 **************************/

// MWMockContext value carrier for mocking authentication in MW
type MWMockContext struct {
	Request *http.Request
}

// MWMocker interface that mocked authentication middleware uses to mock authentication at runtime
type MWMocker interface {
	Mock(MWMockContext) security.Authentication
}

/**************************
	Test Options
 **************************/

type MWMockOptions func(opt *MWMockOption)

type MWMockOption struct {
	Route      web.RouteMatcher
	Condition  web.RequestMatcher
	MWMocker   MWMocker
	Configurer security.Configurer
	Session    bool
}

var defaultMWMockOption = MWMockOption{
	MWMocker: DirectExtractionMWMocker{},
	Route:    matcher.AnyRoute(),
}

// WithMockedMiddleware is a test option that automatically install a middleware that populate/save
// security.Authentication into gin.Context.
//
// This test option works with webtest.WithMockedServer without any additional settings:
// - By default extract security.Authentication from request's context.
// Note: 	Since gin-gonic v1.8.0+, this test option is not required anymore for webtest.WithMockedServer. Values in
//			request's context is automatically linked with gin.Context.
//
// When using with webtest.WithRealServer, a custom MWMocker is required. The MWMocker can be provided by:
// - Using MWCustomMocker option
// - Providing a MWMocker using uber/fx
// - Providing a security.Configurer with NewMockedMW:
// 		<code>
// 		func realServerSecConfigurer(ws security.WebSecurity) {
//			ws.Route(matcher.AnyRoute()).
//				With(NewMockedMW().
//					Mocker(MWMockFunc(realServerMockFunc)),
//				)
// 		}
// 		</code>
// See examples package for more details.
func WithMockedMiddleware(opts ...MWMockOptions) test.Options {
	opt := defaultMWMockOption
	for _, fn := range opts {
		fn(&opt)
	}
	testOpts := []test.Options{
		apptest.WithModules(security.Module),
		apptest.WithFxOptions(
			fx.Invoke(registerSecTest),
		),
	}
	if opt.MWMocker != nil {
		testOpts = append(testOpts, apptest.WithFxOptions(fx.Provide(func() MWMocker { return opt.MWMocker })))
	}
	if opt.Configurer != nil {
		testOpts = append(testOpts, apptest.WithFxOptions(fx.Invoke(func(reg security.Registrar) {
			reg.Register(opt.Configurer)
		})))
	} else {
		testOpts = append(testOpts, apptest.WithFxOptions(fx.Invoke(RegisterTestConfigurer(opts...))))
	}
	if opt.Session {
		testOpts = append(testOpts,
			apptest.WithModules(session.Module),
			apptest.WithFxOptions(fx.Decorate(MockedSessionStoreDecorator)),
		)
	}
	return test.WithOptions(testOpts...)
}

// MWRoute returns option for WithMockedMiddleware.
// This route is applied to the default test security.Configurer
func MWRoute(matchers ...web.RouteMatcher) MWMockOptions {
	return func(opt *MWMockOption) {
		for i, m := range matchers {
			if i == 0 {
				opt.Route = m
			} else {
				opt.Route = opt.Route.Or(m)
			}
		}
	}
}

// MWCondition returns option for WithMockedMiddleware.
// This condition is applied to the default test security.Configurer
func MWCondition(matchers ...web.RequestMatcher) MWMockOptions {
	return func(opt *MWMockOption) {
		for i, m := range matchers {
			if i == 0 {
				opt.Condition = m
			} else {
				opt.Condition = opt.Route.Or(m)
			}
		}
	}
}

// MWEnableSession returns option for WithMockedMiddleware.
// This condition is applied to the default test security.Configurer
func MWEnableSession() MWMockOptions {
	return func(opt *MWMockOption) {
		opt.Session = true
	}
}

// MWCustomConfigurer returns option for WithMockedMiddleware.
// If set to nil, MWMockOption.Route and MWMockOption.Condition are used to generate a default configurer
// If set to non-nil, MWMockOption.Route and MWMockOption.Condition are ignored
func MWCustomConfigurer(configurer security.Configurer) MWMockOptions {
	return func(opt *MWMockOption) {
		opt.Configurer = configurer
	}
}

// MWCustomMocker returns option for WithMockedMiddleware.
// If set to nil, fx provided MWMocker will be used
func MWCustomMocker(mocker MWMocker) MWMockOptions {
	return func(opt *MWMockOption) {
		opt.MWMocker = mocker
	}
}

/**************************
	Mockers
 **************************/

// MWMockFunc wrap a function to MWMocker interface
type MWMockFunc func(MWMockContext) security.Authentication

func (f MWMockFunc) Mock(mc MWMockContext) security.Authentication {
	return f(mc)
}

// DirectExtractionMWMocker is an MWMocker that extracts authentication from context.
// This is the implementation is works together with webtest.WithMockedServer and WithMockedSecurity,
// where a context is injected with security.Authentication and directly passed into http.Request
type DirectExtractionMWMocker struct{}

func (m DirectExtractionMWMocker) Mock(mc MWMockContext) security.Authentication {
	return security.Get(mc.Request.Context())
}

/**************************
	Feature
 **************************/

var (
	FeatureId = security.FeatureId("SecTest", security.FeatureOrderAuthenticator)
)

type regDI struct {
	fx.In
	SecRegistrar security.Registrar `optional:"true"`
}

func registerSecTest(di regDI) {
	if di.SecRegistrar != nil {
		configurer := newFeatureConfigurer()
		di.SecRegistrar.(security.FeatureRegistrar).RegisterFeature(FeatureId, configurer)
	}
}

type Feature struct {
	MWMocker MWMocker
}

// NewMockedMW Standard security.Feature entrypoint, DSL style. Used with security.WebSecurity
func NewMockedMW() *Feature {
	return &Feature{}
}

func (f *Feature) Mocker(mocker MWMocker) *Feature {
	f.MWMocker = mocker
	return f
}

func (f *Feature) MWMockFunc(mocker MWMockFunc) *Feature {
	f.MWMocker = mocker
	return f
}

func (f *Feature) Identifier() security.FeatureIdentifier {
	return FeatureId
}

func Configure(ws security.WebSecurity) *Feature {
	feature := NewMockedMW()
	if fc, ok := ws.(security.FeatureModifier); ok {
		return fc.Enable(feature).(*Feature)
	}
	panic(fmt.Errorf("unable to configure session: provided WebSecurity [%T] doesn't support FeatureModifier", ws))
}

type FeatureConfigurer struct {
}

func newFeatureConfigurer() *FeatureConfigurer {
	return &FeatureConfigurer{}
}

func (c *FeatureConfigurer) Apply(feature security.Feature, ws security.WebSecurity) error {
	f := feature.(*Feature)
	mock := &MockAuthenticationMiddleware{
		MWMocker: f.MWMocker,
	}
	mw := middleware.NewBuilder("mocked-auth-mw").
		Order(security.MWOrderPreAuth + 5).
		Use(mock.AuthenticationHandlerFunc())
	ws.Add(mw)

	return nil
}

/**************************
	Security Configurer
 **************************/

type mwDI struct {
	fx.In
	Registrar security.Registrar `optional:"true"`
	Mocker    MWMocker           `optional:"true"`
}

func RegisterTestConfigurer(opts ...MWMockOptions) func(di mwDI) {
	opt := defaultMWMockOption
	for _, fn := range opts {
		fn(&opt)
	}
	return func(di mwDI) {
		if opt.MWMocker == nil {
			opt.MWMocker = di.Mocker
		}
		configurer := security.ConfigurerFunc(newTestSecurityConfigurer(&opt))
		di.Registrar.Register(configurer)
	}
}

func newTestSecurityConfigurer(opt *MWMockOption) func(ws security.WebSecurity) {
	return func(ws security.WebSecurity) {
		ws = ws.Route(opt.Route).With(NewMockedMW().Mocker(opt.MWMocker))
		if opt.Condition != nil {
			ws.Condition(opt.Condition)
		}
	}
}
