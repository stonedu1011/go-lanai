// Package api Generated by lanai_cli codegen. DO NOT EDIT
// Derived from openapi contract - components
package api

import (
	"github.com/google/uuid"
)

type AdditonalPropertyTest struct {
	AttributeWithEmptyObjAP *AdditonalPropertyTestAttributeWithEmptyObjAP `json:"attributeWithEmptyObjAP"`
	AttributeWithFalseAP    *AdditonalPropertyTestAttributeWithFalseAP    `json:"attributeWithFalseAP"`
	AttributeWithTrueAP     *AdditonalPropertyTestAttributeWithTrueAP     `json:"attributeWithTrueAP"`
}

type AdditonalPropertyTestAttributeWithEmptyObjAP struct {
	Values *map[string]interface{}
}

type AdditonalPropertyTestAttributeWithFalseAP struct {
	Property *string `json:"property"`
}

type AdditonalPropertyTestAttributeWithTrueAP struct {
	Values *map[string]interface{}
}

type ApiPolicy struct {
	Unlimited bool `json:"unlimited"`
}

type Device struct {
	CreatedOn      string              `json:"createdOn" binding:"omitempty,date-time"`
	Id             uuid.UUID           `json:"id"`
	ModifiedOn     *string             `json:"modifiedOn" binding:"omitempty,date-time"`
	ServiceType    *string             `json:"serviceType" binding:"omitempty,max=128"`
	Status         DeviceStatus        `json:"status"`
	StatusDetails  DeviceStatusDetails `json:"statusDetails"`
	SubscriptionId uuid.UUID           `json:"subscriptionId"`
	UserId         uuid.UUID           `json:"userId"`
}

type DeviceStatusDetails struct {
	Values *map[string]DeviceStatus
}

type DeviceStatus struct {
	LastUpdated        string `json:"lastUpdated" binding:"required,date-time"`
	LastUpdatedMessage string `json:"lastUpdatedMessage" binding:"required,min=1,max=128"`
	Severity           string `json:"severity" binding:"required,min=1,max=128"`
	Value              string `json:"value" binding:"required,min=1,max=128"`
}

type GenericObject struct {
	Enabled        GenericObjectEnabled        `json:"enabled"`
	Id             *string                     `json:"id"`
	ValueWithAllOf GenericObjectValueWithAllOf `json:"valueWithAllOf"`
}

type GenericObjectEnabled struct {
	Inner *string `json:"inner"`
}

type GenericObjectValueWithAllOf struct {
	ApiPolicy
}

type GenericResponse struct {
	ArrayOfObjects                  []GenericObject             `json:"arrayOfObjects"`
	ArrayOfRef                      *[]string                   `json:"arrayOfRef"`
	ArrayOfUUIDs                    *[]uuid.UUID                `json:"arrayOfUUIDs"`
	CreatedOnDate                   string                      `json:"createdOnDate" binding:"required,date"`
	CreatedOnDateTime               string                      `json:"createdOnDateTime" binding:"omitempty,date-time"`
	DirectRef                       GenericObject               `json:"directRef"`
	IntegerValue                    *int                        `json:"integerValue" binding:"omitempty,max=5"`
	MyUuid                          uuid.UUID                   `json:"myUuid"`
	NumberArray                     *[]float64                  `json:"numberArray" binding:"omitempty,max=10"`
	NumberValue                     *float64                    `json:"numberValue" binding:"omitempty,max=10"`
	ObjectValue                     *GenericResponseObjectValue `json:"objectValue" binding:"required"`
	StringValue                     *string                     `json:"stringValue" binding:"required,max=128"`
	StringWithEnum                  string                      `json:"stringWithEnum" binding:"omitempty,enumof=asc desc"`
	StringWithNilEnum               *string                     `json:"stringWithNilEnum" binding:"omitempty,enumof=asc desc"`
	StringWithRegexDefinedInFormat  string                      `json:"stringWithRegexDefinedInFormat" binding:"omitempty,regexCD184"`
	StringWithRegexDefinedInPattern string                      `json:"stringWithRegexDefinedInPattern" binding:"required,regexEB33C"`
	Values                          *map[string]string
}

type GenericResponseObjectValue struct {
	ObjectNumber *float64 `json:"objectNumber" binding:"required"`
}

type GenericResponseWithAllOf struct {
	Id *string `json:"id"`
	GenericResponse
}

type TestRequest struct {
	Uuid uuid.UUID `json:"uuid"`
}
