// Package v4 Generated by lanai-cli codegen.
// Derived from contents in openapi contract, path: /my/api/v4/testRequestBodyUsingARefWithAllOf
package v4

import (
	"context"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/web"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/web/rest"
	"cto-github.cisco.com/NFV-BU/test-service/pkg/api"
	"go.uber.org/fx"
)

type TestRequestBodyUsingARefWithAllOfController struct{}

type testRequestBodyUsingARefWithAllOfControllerDI struct {
	fx.In
}

func NewTestRequestBodyUsingARefWithAllOfController(di testRequestBodyUsingARefWithAllOfControllerDI) web.Controller {
	return &TestRequestBodyUsingARefWithAllOfController{}
}

func (c *TestRequestBodyUsingARefWithAllOfController) Mappings() []web.Mapping {
	return []web.Mapping{
		rest.
			New("testrequestbodyusingarefwithallof-post").
			Post("/api/v4/testRequestBodyUsingARefWithAllOf").
			EndpointFunc(c.CreateDevice).
			Build(),
	}
}

func (c *TestRequestBodyUsingARefWithAllOfController) CreateDevice(ctx context.Context, req api.DeviceCreate) (interface{}, error) {
	return nil, nil
}