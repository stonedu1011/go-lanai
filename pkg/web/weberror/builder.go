package weberror

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/web"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/web/matcher"
	"fmt"
)

type MappingBuilder struct {
	name       string
	matcher    web.RouteMatcher
	order      int
	condition     web.RequestMatcher
	translateFunc web.ErrorTranslateFunc
}

func New(name ...string) *MappingBuilder {
	n := "anonymous"
	if len(name) != 0 {
		n = name[0]
	}
	return &MappingBuilder{
		name: n,
		matcher: matcher.AnyRoute(),
	}
}

/*****************************
	Public
******************************/

func (b *MappingBuilder) Name(name string) *MappingBuilder {
	b.name = name
	return b
}

func (b *MappingBuilder) Order(order int) *MappingBuilder {
	b.order = order
	return b
}

func (b *MappingBuilder) With(translator web.ErrorTranslator) *MappingBuilder {
	b.translateFunc = translator.Translate
	return b
}

func (b *MappingBuilder) ApplyTo(matcher web.RouteMatcher) *MappingBuilder {
	b.matcher = matcher
	return b
}

func (b *MappingBuilder) Use(translateFunc web.ErrorTranslateFunc) *MappingBuilder {
	b.translateFunc = translateFunc
	return b
}

func (b *MappingBuilder) WithCondition(condition web.RequestMatcher) *MappingBuilder {
	b.condition = condition
	return b
}

func (b *MappingBuilder) Build() web.ErrorTranslateMapping {
	if b.matcher == nil {
		b.matcher = matcher.AnyRoute()
	}
	if b.name == "" {
		b.name = fmt.Sprintf("%v", b.matcher)
	}
	if b.translateFunc == nil {
		panic(fmt.Errorf("unable to build '%s' error translation mapping: error translate function is required. please use With(...) or Use(...)", b.name))
	}
	return web.NewErrorTranslateMapping(b.name, b.order, b.matcher, b.condition, b.translateFunc)
}

/*****************************
	Helpers
******************************/



