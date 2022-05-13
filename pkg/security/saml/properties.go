package saml

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/bootstrap"
	"github.com/pkg/errors"
)

const SamlPropertiesPrefix = "security.auth.saml"

type SamlProperties struct {
	CertificateFile string `json:"certificate-file"`
	KeyFile string  `json:"key-file"`
	KeyPassword string `json:"key-password"`
	NameIDFormat string `json:"name-id-format"`
}

func NewSamlProperties() *SamlProperties {
	return &SamlProperties{
		//We use this property by default so that the auth request generated by the saml package will not
		//have NameIDFormat by default
		//See saml.nameIDFormat() in github.com/crewjam/saml
		NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified",
	}
}

func BindSamlProperties(ctx *bootstrap.ApplicationContext) SamlProperties {
	props := NewSamlProperties()
	if err := ctx.Config().Bind(props, SamlPropertiesPrefix); err != nil {
		panic(errors.Wrap(err, "failed to bind SamlProperties"))
	}
	return *props
}
