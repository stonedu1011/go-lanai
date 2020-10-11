package web

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
	"net/http"
)

const (
	kGinContextKey = "GinContext"
)

type Registrar struct {
	engine *gin.Engine
	options []httptransport.ServerOption
}

// TODO support customizers
func NewRegistrar(g *gin.Engine) *Registrar {
	return &Registrar{
		engine: g,
		options: []httptransport.ServerOption{
			httptransport.ServerBefore(ginContextExtractor),
		},
	}
}

// Register is the entry point to register Controller, Mapping and other web related objects
// supported parameter type are:
// 	- Controller
//  - EndpointMapping
//  - StaticMapping
//  - MvcMapping
func (r *Registrar) Register(i interface{}) (err error) {
	switch i.(type) {
	case Controller:
		err = r.registerController(i.(Controller))
	case EndpointMapping:
		err = r.registerEndpointMapping(i.(EndpointMapping))
	case MvcMapping:
		err = r.registerMvcMapping(i.(MvcMapping))
	case StaticMapping:
		err = r.registerStaticMapping(i.(StaticMapping))
	default:
		err = errors.New(fmt.Sprintf("unsupported type [%T]", i))
	}
	return
}

func (r *Registrar) registerController(c Controller) (err error) {
	endpoints := c.Endpoints()
	for _, m := range endpoints {
		if err = r.Register(m); err != nil {
			err = fmt.Errorf("invalid endpoint mapping in Controller [%T]: %v", c, err.Error())
			break
		}
	}
	return
}

func (r *Registrar) registerEndpointMapping(m EndpointMapping) error {
	s := httptransport.NewServer(
		m.Endpoint(),
		m.DecodeRequestFunc(),
		m.EncodeResponseFunc(),
		r.options...,
	)

	handlerFunc := MakeGinHandlerFunc(s)
	r.engine.Handle(m.Method(), m.Path(), handlerFunc)
	return nil
}

func (r *Registrar) registerStaticMapping(m StaticMapping) error {
	// TODO handle suffix rewrite, e.g. /path/to/swagger -> /path/to/swagger.html
	r.engine.Static(m.Path(), m.StaticRoot())
	return nil
}

func (r *Registrar) registerMvcMapping(_ MvcMapping) error {
	// TODO finish this. It's needed for login page generation
	return nil
}

/**************************
	first class functions
***************************/

func MakeGinHandlerFunc(s *httptransport.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := context.WithValue(c.Request.Context(), kGinContextKey, c)
		c.Request = c.Request.WithContext(reqCtx)
		s.ServeHTTP(c.Writer, c.Request)
	}
}

func ginContextExtractor(ctx context.Context, r *http.Request) (ret context.Context) {
	if ret = r.Context().Value(kGinContextKey).(context.Context); ret == nil {
		return ctx
	}
	return
}




