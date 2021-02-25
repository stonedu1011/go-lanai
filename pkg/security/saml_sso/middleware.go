package saml_auth

import (
	"context"
	"crypto"
	"crypto/x509"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/utils"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/web"
	"encoding/xml"
	"errors"
	"github.com/crewjam/saml"
	saml_logger "github.com/crewjam/saml/logger"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
)

type Options struct {
	Key                    crypto.PrivateKey
	Cert                   *x509.Certificate
	EntityIdUrl            url.URL
	SsoUrl                 url.URL
	serviceProviderManager SamlClientStore
}

type SamlAuthorizeEndpointMiddleware struct {
	accountStore security.AccountStore

	//used to load the saml clients
	samlClientStore SamlClientStore
	//manages the resolved service provider metadata
	spMetadataManager *SpMetadataManager

	idp               *saml.IdentityProvider

	attributeGenerator AttributeGenerator
}

func NewSamlAuthorizeEndpointMiddleware(opts Options,
	serviceProviderManager SamlClientStore,
	accountStore security.AccountStore,
	attributeGenerator AttributeGenerator) *SamlAuthorizeEndpointMiddleware {

	spDescriptorManager := &SpMetadataManager{
		cache: make(map[SamlSpDetails]*saml.EntityDescriptor),
		httpClient: http.DefaultClient,
	}

	idp := &saml.IdentityProvider{
		Key:                     opts.Key,
		Logger:                  saml_logger.DefaultLogger, //TODO replace with our own logger
		Certificate:             opts.Cert,
		//since we have our own middleware implementation, this value here only serves the purpose of defining the entity id.
		MetadataURL:             opts.EntityIdUrl,
		SSOURL:                  opts.SsoUrl,
	}

	mw := &SamlAuthorizeEndpointMiddleware{
		idp:                idp,
		samlClientStore:    serviceProviderManager,
		spMetadataManager:  spDescriptorManager,
		accountStore: accountStore,
		attributeGenerator: attributeGenerator,
	}

	return mw
}

func (mw *SamlAuthorizeEndpointMiddleware) AuthorizeHandlerFunc(condition web.RequestMatcher) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if matches, err :=condition.MatchesWithContext(ctx, ctx.Request); !matches || err != nil {
			return
		}

		req, err := saml.NewIdpAuthnRequest(mw.idp, ctx.Request)
		if err != nil {
			mw.handleError(ctx, nil, NewSamlInternalError("error decoding authentication request", err))
			return
		}

		if err = UnmarshalRequest(req); err != nil {
			mw.handleError(ctx, nil, err)
			return
		}

		auth, exist := ctx.Get(security.ContextKeySecurity)
		//sanity check
		if !exist {
			mw.handleError(ctx, nil, NewSamlInternalError("no authentication found", err))
			return
		}

		authentication, ok := auth.(security.Authentication)
		//sanity check
		if !ok {
			mw.handleError(ctx, nil, NewSamlInternalError("authentication type is not supported"))
			return
		}
		//sanity check
		if authentication.State() < security.StateAuthenticated {
			mw.handleError(ctx, nil, NewSamlInternalError("session is not authenticated"))
			return
		}

		serviceProviderID := req.Request.Issuer.Value

		// find the service provider metadata
		spDetails, spMetadata, err := mw.spMetadataManager.GetServiceProvider(serviceProviderID)
		if err != nil {
			mw.handleError(ctx, nil, NewSamlInternalError("cannot find service provider metadata"))
			return
		}
		if len(spMetadata.SPSSODescriptors) != 1 {
			mw.handleError(ctx, nil, NewSamlInternalError("expected exactly one SP SSO descriptor in SP metadata"))
			return
		}
		req.ServiceProviderMetadata = spMetadata
		req.SPSSODescriptor = &spMetadata.SPSSODescriptors[0]

		// Check that the ACS URL matches an ACS endpoint in the SP metadata.
		// After this point, we have the endpoint to send back responses whether it's success or false
		if err = DetermineACSEndpoint(req); err != nil {
			mw.handleError(ctx, nil, err)
			return
		}

		if err = ValidateAuthnRequest(req, spDetails, spMetadata); err != nil {
			mw.handleError(ctx, req, err)
			return
		}

		//check tenancy
		client, err := mw.samlClientStore.GetSamlClientById(serviceProviderID)
		if err != nil { //we shouldn't get an error here because we already have the SP's metadata.
			//if an error does occur, it means there's a programming error
			mw.handleError(ctx, nil, NewSamlInternalError("saml client not found", err))
			return
		}
		err = mw.validateTenantRestriction(ctx, client.GetTenantRestrictions(), authentication)
		if err != nil {
			mw.handleError(ctx, req, err)
			return
		}

		if err = MakeAssertion(ctx, req, authentication, mw.attributeGenerator); err != nil {
			mw.handleError(ctx, req, err)
			return
		}

		if err = MakeAssertionEl(req, spDetails.SkipAssertionEncryption); err != nil {
			mw.handleError(ctx, req, err)
			return
		}

		if err = req.WriteResponse(ctx.Writer); err != nil {
			mw.handleError(ctx, nil, NewSamlInternalError("error writing saml response", err))
			return
		}
	}
}

func (mw *SamlAuthorizeEndpointMiddleware) RefreshMetadataHandler(condition web.RequestMatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		if matches, err :=condition.MatchesWithContext(c, c.Request); !matches || err != nil {
			return
		}

		clients := mw.samlClientStore.GetAllSamlClient()
		mw.spMetadataManager.RefreshCache(clients)
	}
}

func (mw *SamlAuthorizeEndpointMiddleware) MetadataHandlerFunc(c *gin.Context) {
	metadata := mw.idp.Metadata()
	var t = true
	//We always want the authentication request to be signed
	//But because this is not supported by the saml package, we set it here explicitly
	metadata.IDPSSODescriptors[0].WantAuthnRequestsSigned = &t
	w := c.Writer
	buf, _ := xml.MarshalIndent(metadata, "", "  ")
	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	w.Header().Set("Content-Disposition", "attachment; filename=metadata.xml")
	_, _ = w.Write(buf)
}

func (mw *SamlAuthorizeEndpointMiddleware) handleError(c *gin.Context, authRequest *saml.IdpAuthnRequest, err error) {
	if !errors.Is(err, security.ErrorTypeSaml) {
		err = NewSamlInternalError("saml sso internal error", err)
	}

	if authRequest != nil {
		c.Set(CtxKeySamlAuthnRequest, authRequest)
	}

	_ = c.Error(err)
	c.Abort()
}


func (mw *SamlAuthorizeEndpointMiddleware) validateTenantRestriction(ctx context.Context, tenantRestriction utils.StringSet, auth security.Authentication) error {
	if len(tenantRestriction) == 0  {
		return nil
	}

	username, err := security.GetUserName(auth)

	if err != nil {
		return NewSamlInternalError("cannot validate tenancy restriction due to unknown username", err)
	}

	acct, err := mw.accountStore.LoadAccountByUsername(ctx, username)

	defaultAcct, ok := acct.(*security.DefaultAccount)

	if !ok {
		return NewSamlInternalError("cannot validate tenancy restriction due to unspported account implementation", defaultAcct)
	}

	//TODO: we need tenant hierarchy to be able to properly check tenant restriction here
	return nil
}