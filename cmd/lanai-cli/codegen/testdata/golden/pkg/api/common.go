// Package api Generated by lanai_cli codegen. DO NOT EDIT
// Derived from openapi contract - components
package api

type GenericObject struct {
	Enabled GenericObjectEnabled `json:"enabled"`
	Id      *string              `json:"id" binding:"omitempty"`
}

type GenericObjectEnabled struct {
	Inner *string `json:"inner" binding:"omitempty"`
}

type GenericResponse struct {
	ArrayOfObjects                  []GenericObject             `json:"arrayOfObjects"`
	ArrayOfRef                      *[]string                   `json:"arrayOfRef" binding:"omitempty"`
	CreatedOnDate                   string                      `json:"createdOnDate" binding:"required,date"`
	CreatedOnDateTime               string                      `json:"createdOnDateTime" binding:"date-time"`
	DirectRef                       GenericObject               `json:"directRef"`
	IntegerValue                    *int                        `json:"integerValue" binding:"max=5,omitempty"`
	MyUuid                          string                      `json:"myUuid" binding:"uuid"`
	NumberArray                     *[]float64                  `json:"numberArray" binding:"max=10,omitempty"`
	NumberValue                     *float64                    `json:"numberValue" binding:"max=10,omitempty"`
	ObjectValue                     *GenericResponseObjectValue `json:"objectValue" binding:"required"`
	StringValue                     *string                     `json:"stringValue" binding:"required,max=128"`
	StringWithEnum                  string                      `json:"stringWithEnum" binding:"omitempty,enumof=asc desc"`
	StringWithNilEnum               *string                     `json:"stringWithNilEnum" binding:"omitempty,enumof=asc desc"`
	StringWithRegexDefinedInFormat  string                      `json:"stringWithRegexDefinedInFormat" binding:"regexCD184"`
	StringWithRegexDefinedInPattern string                      `json:"stringWithRegexDefinedInPattern" binding:"required,regexEB33C"`
	Values                          *map[string]string
}

type GenericResponseObjectValue struct {
	ObjectNumber *float64 `json:"objectNumber" binding:"required"`
}

type GenericResponseWithAllOf struct {
	Id *string `json:"id" binding:"omitempty"`
	GenericResponse
}

type TestRequest struct {
	Uuid string `json:"uuid" binding:"uuid"`
}

type PathParam struct {
	Scope string `uri:"scope" binding:"required,regexA397E"`
}

type QueryParam struct {
	TestParam *string `form:"testParam" binding:"max=128,omitempty"`
}
