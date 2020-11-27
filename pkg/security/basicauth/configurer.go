package basicauth

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/web/middleware"
	"fmt"
)

const (
	FeatureId = "BasicAuth"
)

// We currently don't have any stuff to configure
//goland:noinspection GoNameStartsWithPackageName
type BasicAuthFeature struct {
	// TODO we may want to override authenticator and other stuff
}

// Standard security.Feature entrypoint
func (f *BasicAuthFeature) Identifier() security.FeatureIdentifier {
	return FeatureId
}

func Configure(ws security.WebSecurity) *BasicAuthFeature {
	feature := New()
	if fc, ok := ws.(security.FeatureModifier); ok {
		_ = fc.Enable(feature) // we ignore error here
		return feature
	}
	panic(fmt.Errorf("unable to configure session: provided WebSecurity [%T] doesn't support FeatureModifier", ws))
}

// Standard security.Feature entrypoint, DSL style. Used with security.WebSecurity
func New() *BasicAuthFeature {
	return &BasicAuthFeature{}
}

//goland:noinspection GoNameStartsWithPackageName
type BasicAuthConfigurer struct {

}

func newBasicAuthConfigurer() *BasicAuthConfigurer {
	return &BasicAuthConfigurer{
	}
}

func (bac *BasicAuthConfigurer) Apply(_ security.Feature, ws security.WebSecurity) error {
	// TODO
	basicAuth := NewBasicAuthMiddleware(ws.Authenticator())
	auth := middleware.NewBuilder("basic auth").
		Order(security.MWOrderBasicAuth).
		Use(basicAuth.HandlerFunc())

	ws.Add(auth)
	return nil
}