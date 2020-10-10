package rest

import (
	"errors"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
)

type MappingBuilder interface {
	Path(string) MappingBuilder
	Method(string) MappingBuilder
	EndpointFunc(EndpointFunc) MappingBuilder

	// Overrides
	Endpoint(endpoint.Endpoint) MappingBuilder
	DecodeRequestFunc(httptransport.DecodeRequestFunc) MappingBuilder
	EncodeRequestFunc(httptransport.EncodeRequestFunc) MappingBuilder
	DecodeResponseFunc(httptransport.DecodeResponseFunc) MappingBuilder
	EncodeResponseFunc(httptransport.EncodeResponseFunc) MappingBuilder

	// Builder
	Build() *endpointMapping
}

type mappingBuilder struct {
	path         string
	method       string
	endpointFunc EndpointFunc
	endpoint           endpoint.Endpoint
	decodeRequestFunc  httptransport.DecodeRequestFunc
	encodeRequestFunc  httptransport.EncodeRequestFunc
	decodeResponseFunc httptransport.DecodeResponseFunc
	encodeResponseFunc httptransport.EncodeResponseFunc
}

func NewBuilder() MappingBuilder {
	return &mappingBuilder{}
}

/*****************************
	MappingBuilder Impl
******************************/
func (b *mappingBuilder) Path(path string) MappingBuilder {
	b.path = path
	return b
}

func (b *mappingBuilder) Method(method string) MappingBuilder {
	b.method = method
	return b
}

func (b *mappingBuilder) EndpointFunc(endpointFunc EndpointFunc) MappingBuilder {
	b.endpointFunc = endpointFunc
	return b
}

// Overrides
func (b *mappingBuilder) Endpoint(endpoint endpoint.Endpoint) MappingBuilder {
	b.endpoint = endpoint
	return b
}

func (b *mappingBuilder) DecodeRequestFunc(f httptransport.DecodeRequestFunc) MappingBuilder {
	b.decodeRequestFunc = f
	return b
}

func (b *mappingBuilder) EncodeRequestFunc(f httptransport.EncodeRequestFunc) MappingBuilder {
	b.encodeRequestFunc = f
	return b
}

func (b *mappingBuilder) DecodeResponseFunc(f httptransport.DecodeResponseFunc) MappingBuilder {
	b.decodeResponseFunc = f
	return b
}

func (b *mappingBuilder) EncodeResponseFunc(f httptransport.EncodeResponseFunc) MappingBuilder {
	b.encodeResponseFunc = f
	return b
}

func (b *mappingBuilder) Build() *endpointMapping {
	if err := b.validate(); err != nil {
		panic(err)
	}
	return b.buildMapping()
}

// TODO more validation and better error handling
func (b *mappingBuilder) validate() (err error) {
	if b.path == "" || b.method == "" {
		err = errors.New("empty Path")
	}
	return
}

func (b *mappingBuilder) buildMapping() *endpointMapping {
	m := &endpointMapping {
		path: b.path,
		method: b.method,
		endpointFunc: b.endpointFunc,
		endpoint: nil,
		decodeRequestFunc: httptransport.NopRequestDecoder,
		encodeRequestFunc: GenericEncodeRequestFunc,
		decodeResponseFunc: nil, // TODO
		encodeResponseFunc: GenericEncodeResponseFunc,
	}

	if b.endpointFunc != nil {
		metadata := MakeEndpointFuncMetadata(b.endpointFunc)
		m.endpoint = MakeEndpoint(metadata)
		m.decodeRequestFunc = MakeGinBindingDecodeRequestFunc(metadata)
		m.decodeResponseFunc = nil // TODO
	}

	b.customize(m)
	return m
}

func (b *mappingBuilder) customize(m *endpointMapping) {
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
