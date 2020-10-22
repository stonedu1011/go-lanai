package rest

import (
	"cto-github.cisco.com/livdu/jupiter/pkg/web"
	"errors"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"net/http"
)

// EndpointFunc is a function with following signature
// 	- two input parameters with 1st as context.Context and 2nd as <request>
// 	- two output parameters with 1st as <response> and 2nd as error
// where
// <request>:   a struct or a pointer to a struct whose fields are properly tagged
// <response>:  a struct or a pointer to a struct whose fields are properly tagged.
// 				if decoding is not supported (rest not used by any go client), it can be an interface{}
// e.g.: func(context.Context, request *AnyStructWithTag) (response *AnyStructWithTag, error) {...}
type EndpointFunc web.MvcHandlerFunc

type MappingBuilder struct {
	name               string
	path               string
	method             string
	endpointFunc       EndpointFunc
	endpoint           endpoint.Endpoint
	decodeRequestFunc  httptransport.DecodeRequestFunc
	encodeRequestFunc  httptransport.EncodeRequestFunc
	decodeResponseFunc httptransport.DecodeResponseFunc
	encodeResponseFunc httptransport.EncodeResponseFunc
}

func NewBuilder(names ...string) *MappingBuilder {
	name := "unknown"
	if len(names) > 0 {
		name = names[0]
	}
	return &MappingBuilder{
		name: name,
	}
}

/*****************************
	Public
******************************/
func (b *MappingBuilder) Name(name string) *MappingBuilder {
	b.name = name
	return b
}
func (b *MappingBuilder) Path(path string) *MappingBuilder {
	b.path = path
	return b
}

func (b *MappingBuilder) Method(method string) *MappingBuilder {
	b.method = method
	return b
}

func (b *MappingBuilder) EndpointFunc(endpointFunc EndpointFunc) *MappingBuilder {
	b.endpointFunc = endpointFunc
	return b
}

// Convenient setters
func (b *MappingBuilder) Get(path string) *MappingBuilder {
	return b.Path(path).Method(http.MethodGet)
}

func (b *MappingBuilder) Post(path string) *MappingBuilder {
	return b.Path(path).Method(http.MethodPost)
}

func (b *MappingBuilder) Put(path string) *MappingBuilder {
	return b.Path(path).Method(http.MethodPut)
}

func (b *MappingBuilder) Patch(path string) *MappingBuilder {
	return b.Path(path).Method(http.MethodPatch)
}

func (b *MappingBuilder) Delete(path string) *MappingBuilder {
	return b.Path(path).Method(http.MethodDelete)
}

func (b *MappingBuilder) Options(path string) *MappingBuilder {
	return b.Path(path).Method(http.MethodOptions)
}

func (b *MappingBuilder) Head(path string) *MappingBuilder {
	return b.Path(path).Method(http.MethodHead)
}

// Overrides
func (b *MappingBuilder) Endpoint(endpoint endpoint.Endpoint) *MappingBuilder {
	b.endpoint = endpoint
	return b
}

func (b *MappingBuilder) DecodeRequestFunc(f httptransport.DecodeRequestFunc) *MappingBuilder {
	b.decodeRequestFunc = f
	return b
}

func (b *MappingBuilder) EncodeRequestFunc(f httptransport.EncodeRequestFunc) *MappingBuilder {
	b.encodeRequestFunc = f
	return b
}

func (b *MappingBuilder) DecodeResponseFunc(f httptransport.DecodeResponseFunc) *MappingBuilder {
	b.decodeResponseFunc = f
	return b
}

func (b *MappingBuilder) EncodeResponseFunc(f httptransport.EncodeResponseFunc) *MappingBuilder {
	b.encodeResponseFunc = f
	return b
}

func (b *MappingBuilder) Build() web.EndpointMapping {
	if err := b.validate(); err != nil {
		panic(err)
	}
	return b.buildMapping()
}

/*****************************
	Private
******************************/
type mapping struct {
	endpoint           endpoint.Endpoint
	decodeRequestFunc  httptransport.DecodeRequestFunc
	encodeRequestFunc  httptransport.EncodeRequestFunc
	decodeResponseFunc httptransport.DecodeResponseFunc
	encodeResponseFunc httptransport.EncodeResponseFunc
}

// TODO more validation and better error handling
func (b *MappingBuilder) validate() (err error) {
	if b.path == "" || b.method == "" {
		err = errors.New("empty Path")
	}
	return
}

func (b *MappingBuilder) buildMapping() web.MvcMapping {
	m := &mapping{
		decodeRequestFunc:  httptransport.NopRequestDecoder,
		encodeRequestFunc:  jsonEncodeRequestFunc,
		decodeResponseFunc: nil, // TODO
		encodeResponseFunc: jsonEncodeResponseFunc,
	}

	if b.endpointFunc != nil {
		metadata := web.MakeFuncMetadata(b.endpointFunc, nil)
		m.endpoint = web.MakeEndpoint(metadata)
		m.decodeRequestFunc = web.MakeGinBindingDecodeRequestFunc(metadata)
	}

	b.customize(m)
	return web.NewMvcMapping(b.name, b.path, b.method,
		m.endpoint, m.decodeRequestFunc, m.encodeRequestFunc,
		m.decodeResponseFunc, m.encodeResponseFunc,
		jsonErrorEncoder)
}

func (b *MappingBuilder) customize(m *mapping) {
	if b.endpoint != nil {
		m.endpoint = b.endpoint
	}

	if b.encodeRequestFunc != nil {
		m.encodeRequestFunc = b.encodeRequestFunc
	}

	if b.decodeRequestFunc != nil {
		m.decodeRequestFunc = b.decodeRequestFunc
	}

	if b.encodeResponseFunc != nil {
		m.encodeResponseFunc = b.encodeResponseFunc
	}

	if b.decodeResponseFunc != nil {
		m.decodeResponseFunc = b.decodeResponseFunc
	}
}