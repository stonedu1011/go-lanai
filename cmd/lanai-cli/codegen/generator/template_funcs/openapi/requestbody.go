package openapi

import (
	"github.com/getkin/kin-openapi/openapi3"
	"path"
)

type RequestBody openapi3.RequestBodyRef

func (r RequestBody) ContainsRef() (result bool) {
	if r.Ref != "" {
		return true
	}
	if r.Value == nil {
		return false
	}
	for _, j := range r.Value.Content {
		if j.Schema.Ref != "" {
			result = true
		}
	}
	return result
}

func (r RequestBody) CountFields() (result int) {
	if r.Ref != "" {
		result++
	}
	if r.Value != nil {
		result += len(r.Value.Content)
	}
	return result
}

func (r RequestBody) RefsUsed() (result []string) {
	if r.CountFields() == 0 {
		return
	}
	if r.Ref != "" {
		result = append(result, path.Base(r.Ref))
	}

	if r.Value == nil {
		return
	}
	//Assumption - Responses will have just one mediatype defined in contract, e.g just "application/json"
	if len(r.Value.Content) > 1 {
		logger.Warn("more than one mediatype defined in requestBody, undefined behavior may occur")
	}
	for _, schema := range r.Schemas() {
		if schema.Ref != "" {
			result = append(result, path.Base(schema.Ref))
		} else if schema.Value.Type == "array" && schema.Value.Items.Ref != "" {
			result = append(result, schema.Value.Items.Ref)
		}
	}
	return result
}

func (r RequestBody) Schemas() (result openapi3.SchemaRefs) {
	for _, c := range r.Value.Content {
		result = append(result, SchemaRef(*c.Schema).AllSchemas()...)
	}
	return result
}