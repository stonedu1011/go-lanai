package security

import (
	"fmt"
	"net/url"
	"strings"
)

type UrlBuilderOptions func(opt *UrlBuilderOption)

type UrlBuilderOption struct {
	Subdomain string
	Path      string
}

type Issuer interface {
	// Basic informations
	Protocol() string
	Domain() string
	Port() int
	ContextPath() string
	IsSecured() bool

	// Identifier is the unique identifier of the deployed auth server
	// Typeical implementation is to use base url of issuer's domain.
	Identifier() string

	// LevelOfAssurance construct level-of-assurance string with given string
	// level-of-assurance represent how confident the auth issuer is about user's identity
	// ref: https://it.wisc.edu/about/user-authentication-and-levels-of-assurance/
	LevelOfAssurance(level int) string

	// BuildUrl build a URL with given url builder options
	// Implementation specs:
	// 	1. if UrlBuilderOption.Subdomain is not specified, Issuer.Domain() should be used
	//  2. if UrlBuilderOption.Subdomain is not a subdomain of Issuer.Domain(), an error should be returned
	//  3. should assume UrlBuilderOption.Path doesn't includes Issuer.ContextPath and the generated URL always
	//	   include Issuer.ContextPath
	//  4. if UrlBuilderOption.Path is not specified, the generated URL could be used as a base URL
	//	5. BuildUrl should not returns error when no options provided
	BuildUrl(...UrlBuilderOptions) (*url.URL, error)
}

/***************************
	Default Impl.
 ***************************/
type DefaultIssuerDetails struct {
	Protocol    string
	Domain      string
	Port        int
	ContextPath string
	IncludePort bool
}

type DefaultIssuer struct {
	DefaultIssuerDetails
}

func NewIssuer(opts ...func(*DefaultIssuerDetails)) *DefaultIssuer {
	opt := DefaultIssuerDetails{

	}
	for _, f := range opts {
		f(&opt)
	}
	return &DefaultIssuer{
		DefaultIssuerDetails: opt,
	}
}

func (i DefaultIssuer) Protocol() string {
	return i.DefaultIssuerDetails.Protocol
}

func (i DefaultIssuer) Domain() string {
	return i.DefaultIssuerDetails.Domain
}

func (i DefaultIssuer) Port() int {
	return i.DefaultIssuerDetails.Port
}

func (i DefaultIssuer) ContextPath() string {
	return i.DefaultIssuerDetails.ContextPath
}

func (i DefaultIssuer) IsSecured() bool {
	return strings.ToLower(i.DefaultIssuerDetails.Protocol) == "https"
}

func (i DefaultIssuer) Identifier() string {
	id, _ := i.BuildUrl()
	return id.String()
}

func (i DefaultIssuer) LevelOfAssurance(level int) string {
	path := fmt.Sprintf("/loa-%d", level)
	loa, _ := i.BuildUrl(func(opt *UrlBuilderOption) {
		opt.Path = path
	})
	return loa.String()
}

func (i DefaultIssuer) BuildUrl(options ...UrlBuilderOptions) (*url.URL, error) {
	opt := UrlBuilderOption{}
	for _, f := range options {
		f(&opt)
	}
	if opt.Subdomain == "" {
		opt.Subdomain = i.DefaultIssuerDetails.Domain
	}

	if strings.HasSuffix(opt.Subdomain, i.DefaultIssuerDetails.Domain) && strings.HasPrefix(opt.Subdomain, ".") {
		return nil, fmt.Errorf("invalid subdomain %s", opt.Subdomain)
	}

	ret := &url.URL{}
	ret.Scheme = i.DefaultIssuerDetails.Protocol
	ret.Host = opt.Subdomain
	if i.DefaultIssuerDetails.IncludePort {
		ret.Host = fmt.Sprintf("%s:%d", ret.Host, i.DefaultIssuerDetails.Port)
	}

	ret.Path = i.DefaultIssuerDetails.ContextPath
	if opt.Path != "" {
		ret = ret.ResolveReference(&url.URL{Path: opt.Path})
	}

	return ret, nil
}
