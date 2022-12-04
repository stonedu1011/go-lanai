package internal

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/cmd/lanai-cli/codegen/generator/internal/representation"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"regexp"
	"strings"
	"text/template"
)

var (
	structsFuncMap = template.FuncMap{
		"propertyType":        propertyType,
		"structTag":           structTag,
		"requiredList":        requiredList,
		"containsSingularRef": containsSingularRef,
		"shouldHavePointer":   shouldHavePointer,
		"defaultNameFromPath": defaultNameFromPath,
		"registerStruct":      registerStruct,
		"structLocation":      structLocation,
		"importsUsedByPath":   importsUsedByPath,
		"isEmpty":             isEmpty,
		"property":            representation.NewProperty,
		"propertyTypePrefix":  representation.PropertyTypePrefix,
		"operation":           representation.NewOperation,
		"schema":              representation.NewSchema,
		"pathOperations":      pathOperations,
	}
)

func requiredList(val interface{}) ([]string, error) {
	var list []string
	interfaceType := getInterfaceType(val)
	switch interfaceType {
	case SchemaRefPtr:
		list = val.(*openapi3.SchemaRef).Value.Required
	case ParameterPtr:
		parameter := val.(*openapi3.Parameter)
		if parameter.Required {
			list = append(list, parameter.Name)
		}
	default:
		return nil, fmt.Errorf("requiredList error: unsupported interface %v", interfaceType)
	}
	return list, nil
}

func defaultNameFromPath(val string) string {
	result := val
	parts := regexp.MustCompile(".+\\/(v\\d+)\\/(.+)").FindStringSubmatch(val)
	if len(parts) == 3 {
		result = parts[2]
	}
	result = strings.ReplaceAll(result, "/", "")
	result = strings.ReplaceAll(result, "{", "")
	result = strings.ReplaceAll(result, "}", "")

	return result
}

var structRegistry = make(map[string]string)

func registerStruct(schemaName string, packageName string) string {
	structRegistry[strings.ToLower(schemaName)] = packageName
	return ""
}

func structLocation(schemaName string) string {
	return structRegistry[strings.ToLower(schemaName)]
}
