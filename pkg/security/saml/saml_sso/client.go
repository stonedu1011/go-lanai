package saml_auth

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/utils"
)

type SamlClient interface {
	GetEntityId() string
	GetMetadataSource() string
	ShouldSkipAssertionEncryption() bool
	ShouldSkipAuthRequestSignatureVerification() bool
	GetTenantRestrictions() utils.StringSet

	ShouldMetadataRequireSignature() bool
	ShouldMetadataTrustCheck() bool
	GetMetadataTrustedKeys() []string
}

type DefaultSamlClient struct {
	SamlSpDetails
	TenantRestrictions utils.StringSet
}

func (c DefaultSamlClient) ShouldMetadataRequireSignature() bool {
	return c.MetadataRequireSignature
}

func (c DefaultSamlClient) ShouldMetadataTrustCheck() bool {
	return c.MetadataTrustCheck
}

func (c DefaultSamlClient) GetMetadataTrustedKeys() []string {
	return c.MetadataTrustedKeys
}

func (c DefaultSamlClient) GetEntityId() string {
	return c.EntityId
}

func (c DefaultSamlClient) GetMetadataSource() string {
	return c.MetadataSource
}

func (c DefaultSamlClient) ShouldSkipAssertionEncryption() bool {
	return c.SkipAssertionEncryption
}

func (c DefaultSamlClient) ShouldSkipAuthRequestSignatureVerification() bool {
	return c.SkipAuthRequestSignatureVerification
}

func (c DefaultSamlClient) GetTenantRestrictions() utils.StringSet {
	return c.TenantRestrictions
}

type SamlSpDetails struct {
	EntityId string
	MetadataSource string
	SkipAssertionEncryption bool
	SkipAuthRequestSignatureVerification bool

	MetadataRequireSignature bool
	MetadataTrustCheck bool
	MetadataTrustedKeys []string

	SecurityProfile string //TODO: at the moment only support metaiop profile (or no profile - which defaults to metaiop) https://docs.spring.io/autorepo/docs/spring-security-saml/1.0.x-SNAPSHOT/reference/htmlsingle/#configuration-security-profiles-pkix
}

type SamlClientStore interface {
	GetAllSamlClient() []SamlClient
	GetSamlClientById(id string) (SamlClient, error)
}