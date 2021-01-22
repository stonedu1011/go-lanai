package samllogin

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/redirect"
	"encoding/base64"
	"encoding/xml"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
)

/**
	A SAML service provider should be able to work with multiple identity providers.
	Because the saml package assumes a service provider is configured with one idp only,
	we use the internal field to store information about this service provider,
	and we will create new saml.ServiceProvider struct for each new IDP connection when its needed.
 */
type ServiceProviderMiddleware struct {
	//using value instead of pointer here because we need to copy it when connecting to specific idps.
	// the methods on saml.ServiceProvider are actually pointer receivers. golang will implicitly use
	// the pointers to these value as receivers
	internal saml.ServiceProvider
	idpManager IdentityProviderManager

	// list of bindings, can be saml.HTTPPostBinding or saml.HTTPRedirectBinding
	// order indicates preference
	bindings       []string
	requestTracker samlsp.RequestTracker

	authenticator security.Authenticator
	successHandler security.AuthenticationSuccessHandler

	fallbackEntryPoint security.AuthenticationEntryPoint

	clientManager *CacheableIdpClientManager
}

type Options struct {
	URL               url.URL
	Key               *rsa.PrivateKey
	Certificate       *x509.Certificate
	Intermediates     []*x509.Certificate
	ACSPath	string
	MetadataPath string
	SLOPath string
	AllowIDPInitiated bool
	SignRequest       bool
	ForceAuthn        bool
}


func NewMiddleware(sp saml.ServiceProvider, tracker samlsp.RequestTracker,
	idpManager IdentityProviderManager,
	clientManager *CacheableIdpClientManager,
	handler security.AuthenticationSuccessHandler, authenticator security.Authenticator,
	errorPath string) *ServiceProviderMiddleware {

	return &ServiceProviderMiddleware{
		internal:           sp,
		bindings:           []string{saml.HTTPPostBinding, saml.HTTPRedirectBinding},
		idpManager:         idpManager,
		clientManager:      clientManager,
		requestTracker:     tracker,
		successHandler:     handler,
		authenticator:      authenticator,
		fallbackEntryPoint: redirect.NewRedirectWithRelativePath(errorPath),
	}
}

func (sp *ServiceProviderMiddleware) MetadataHandlerFunc(c *gin.Context) {
	index := 0
	descriptor := sp.internal.Metadata()
	acs := descriptor.SPSSODescriptors[0].AssertionConsumerServices[0]
	t := true
	acs.IsDefault = &t
	acs.Index = index
	mergedAcs := []saml.IndexedEndpoint{acs}

	//we don't support single logout yet, so don't include this in metadata
	descriptor.SPSSODescriptors[0].SingleLogoutServices = nil

	for _, delegate := range sp.clientManager.GetAllClients() {
		index++
		delegateDescriptor := delegate.Metadata().SPSSODescriptors[0]
		delegateAcs := delegateDescriptor.AssertionConsumerServices[0]
		delegateAcs.Index = index
		mergedAcs = append(mergedAcs, delegateAcs)
	}

	descriptor.SPSSODescriptors[0].AssertionConsumerServices = mergedAcs
	
	w := c.Writer
	buf, _ := xml.MarshalIndent(descriptor, "", "  ")
	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	w.Header().Set("Content-Disposition", "attachment; filename=metadata.xml")
	_, _ = w.Write(buf)
}

func (sp *ServiceProviderMiddleware) MakeAuthenticationRequest(idpIdentifier string, r *http.Request, w http.ResponseWriter) error {
	client, ok := sp.clientManager.GetClientByDomain(idpIdentifier)

	if !ok {
		return security.NewInternalAuthenticationError("cannot find idp for this domain")
	}

	var bindingLocation string
	var binding string
	for _, b := range sp.bindings {
		bindingLocation = client.GetSSOBindingLocation(b)
		if bindingLocation != "" {
			binding = b
			break
		}
	}

	if bindingLocation == "" {
		return security.NewInternalAuthenticationError("IDP does not have supported bindings.")
	}

	authReq, err := client.MakeAuthenticationRequest(bindingLocation)

	if err != nil {
		return security.NewInternalAuthenticationError("cannot make auth request to binding location", err)
	}

	relayState, err := sp.requestTracker.TrackRequest(w, r, authReq.ID)
	if err != nil {
		return security.NewInternalAuthenticationError("cannot track saml auth request", err)
	}

	if binding == saml.HTTPRedirectBinding {
		redirectURL := authReq.Redirect(relayState)
		w.Header().Add("Location", redirectURL.String())
		w.WriteHeader(http.StatusFound)
	} else if binding == saml.HTTPPostBinding {
		//TODO: we can control the authReq.POST method so that it's not separate in two places.
		w.Header().Add("Content-Security-Policy", ""+
			"default-src; "+
			"script-src 'sha256-AjPdJSbZmeWHnEc5ykvJFay8FTWeTeRbs9dutfZ0HqE='; "+ //this hash matches the inline script generated by authReq.Post
			"reflected-xss block; referrer no-referrer;")
		w.Header().Add("Content-type", "text/html")

		body := append([]byte(`<!DOCTYPE html><html><body>`), authReq.Post(relayState)...)
		body = append(body, []byte(`</body></html>`)...)
		_, err = w.Write(body)
		if err != nil {
			return security.NewInternalAuthenticationError("cannot post auth request", err)
		}
	}
	return nil
}

func (sp *ServiceProviderMiddleware) ACSHandlerFunc(c *gin.Context) {
	r := c.Request
	err := r.ParseForm()
	if err != nil {
		sp.handleError(c, err)
		return
	}

	//Parse the response and get entityId
	rawResponseBuf, err := base64.StdEncoding.DecodeString(r.PostForm.Get("SAMLResponse"))
	if err != nil {
		sp.handleError(c, err)
		return
	}

	// do some validation first before we decrypt
	resp := saml.Response{}
	if err := xml.Unmarshal(rawResponseBuf, &resp); err != nil {
		sp.handleError(c, err)
		return
	}

	delegate, ok := sp.clientManager.GetClientByEntityId(resp.Issuer.Value)
	if !ok {
		sp.handleError(c, security.NewInternalAuthenticationError("cannot find idp metadata corresponding for assertion"))
		return
	}

	var possibleRequestIDs []string
	if sp.internal.AllowIDPInitiated {
		possibleRequestIDs = append(possibleRequestIDs, "")
	}

	trackedRequests := sp.requestTracker.GetTrackedRequests(r)
	for _, tr := range trackedRequests {
		possibleRequestIDs = append(possibleRequestIDs, tr.SAMLRequestID)
	}

	assertion, err := delegate.ParseResponse(r, possibleRequestIDs)
	if err != nil {
		sp.handleError(c, security.NewInternalAuthenticationError("error processing assertion", err))
		return
	}

	candidate := &AssertionCandidate{
		Assertion: assertion,
	}
	auth, err := sp.authenticator.Authenticate(c, candidate)

	if err != nil {
		sp.handleError(c, err)
		return
	}

	before := security.Get(c)
	sp.handleSuccess(c, before, auth)

}

//cache that are populated by the refresh metadata middleware instead of populated dynamically on commence
// because in a multi-instance micro service deployment, the auth request and auth response can occur on
// different instance
func (sp *ServiceProviderMiddleware) RefreshMetadataHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		sp.clientManager.RefreshCache(sp.idpManager.GetAllIdentityProvider())
	}
}

func (sp *ServiceProviderMiddleware) Commence(c context.Context, r *http.Request, w http.ResponseWriter, _ error) {
	//TODO: extract the domain related condition out so that it can be anything
	err := sp.MakeAuthenticationRequest(r.Host, r, w)
	if err != nil {
		sp.fallbackEntryPoint.Commence(c, r, w, err)
	}
}

func (sp *ServiceProviderMiddleware) handleSuccess(c *gin.Context, before, new security.Authentication) {
	if new != nil {
		c.Set(gin.AuthUserKey, new.Principal())
		c.Set(security.ContextKeySecurity, new)
	}
	sp.successHandler.HandleAuthenticationSuccess(c, c.Request, c.Writer, before, new)
	if c.Writer.Written() {
		c.Abort()
	}
}

func (sp *ServiceProviderMiddleware) handleError(c *gin.Context, err error) {
	security.Clear(c)
	_ = c.Error(err)
	c.Abort()
}