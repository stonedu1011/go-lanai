package types

import (
	"context"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/data/types/pqx"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/log"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/tenancy"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/utils"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/utils/reflectutils"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"reflect"
)

var logger = log.New("DB.Tenancy")

const (
	fieldTenantID   = "TenantID"
	fieldTenantPath = "TenantPath"
	colTenantID     = "tenant_id"
	colTenantPath   = "tenant_path"
)

var (
	typeUUID          = reflect.TypeOf(uuid.Nil)
	typeUUIDArr       = reflect.TypeOf(pqx.UUIDArray{})
	mapKeysTenantID   = utils.NewStringSet(fieldTenantID, colTenantID)
	mapKeysTenantPath = utils.NewStringSet(fieldTenantPath, colTenantPath)
)

type Tenancy struct {
	TenantID   uuid.UUID     `gorm:"type:KeyID;not null"`
	TenantPath pqx.UUIDArray `gorm:"type:uuid[];index:,type:gin;not null"  json:"-"`
}

func (t *Tenancy) BeforeCreate(tx *gorm.DB) error {
	//if tenantId is not available
	if t.TenantID == uuid.Nil {
		return errors.New("tenantId is required")
	}

	if !security.HasAccessToTenant(tx.Statement.Context, t.TenantID.String()) {
		return errors.New(fmt.Sprintf("user does not have access to tenant %s", t.TenantID.String()))
	}

	path, err := tenancy.GetTenancyPath(tx.Statement.Context, t.TenantID.String())

	if err != nil {
		return err
	}

	t.TenantPath = path

	return nil
}

// BeforeUpdate Check if user is allowed to update this item's tenancy to the target tenant.
// (i.e. if user has access to the target tenant)
// We don't check the original tenancy because we don't have that information in this hook. That check has to be done
// in application code.
func (t *Tenancy) BeforeUpdate(tx *gorm.DB) error {
	dest := tx.Statement.Dest
	tenantId, e := t.extractTenantId(tx.Statement.Context, dest)
	if e != nil || tenantId == uuid.Nil {
		return e
	}

	logger.WithContext(tx.Statement.Context).Debugf("target tenancy is %v", tenantId)
	if !security.HasAccessToTenant(tx.Statement.Context, tenantId.String()) {
		return errors.New(fmt.Sprintf("user does not have access to tenant %s", t.TenantID.String()))
	}

	path, e := tenancy.GetTenancyPath(tx.Statement.Context, tenantId.String())
	if e != nil {
		return e
	}

	return t.updateTenantPath(tx.Statement.Context, dest, path)
}

func (t *Tenancy) BeforeDelete(tx *gorm.DB) error {
	//TODO: possible to add where clause to query?

	//TODO: if tenantId changed, check if user has access to target tenantId

	return nil
}

func (t *Tenancy) extractTenantId(_ context.Context, dest interface{}) (uuid.UUID, error) {
	v := reflect.ValueOf(dest)
	for ; v.Kind() == reflect.Ptr; v = v.Elem() {
	}

	switch v.Kind() {
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return uuid.Nil, fmt.Errorf("unsupported gorm update target type [%T], please use struct ptr, struct or map", dest)
		}
		if _, ev, ok := t.findMapValue(v, mapKeysTenantID, typeUUID); ok {
			return ev.Interface().(uuid.UUID), nil
		}
	case reflect.Struct:
		_, fv, ok := t.findStructField(v, fieldTenantID, typeUUID)
		if ok {
			return fv.Interface().(uuid.UUID), nil
		}
	default:
		return uuid.Nil, fmt.Errorf("unsupported gorm update target type [%T], please use struct ptr, struct or map", dest)
	}
	return uuid.Nil, nil
}

func (t *Tenancy) updateTenantPath(_ context.Context, dest interface{}, tenancyPath pqx.UUIDArray) error {
	v := reflect.ValueOf(dest)
	if v.Kind() == reflect.Struct {
		return fmt.Errorf("cannot update tenancy automatically to %T, please use struct ptr or map", dest)
	}
	for ; v.Kind() == reflect.Ptr; v = v.Elem() {
	}

	switch v.Kind() {
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("cannot update tenancy automatically with gorm update target type [%T], please use struct ptr or map", dest)
		}
		ek, ev, ok := t.findMapValue(v, mapKeysTenantPath, typeUUIDArr)
		switch {
		case ok && !reflect.DeepEqual(ev.Interface(), tenancyPath):
			return fmt.Errorf("incorrect %s was set to gorm update target map", ek)
		case !ok:
			v.SetMapIndex(reflect.ValueOf(fieldTenantPath), reflect.ValueOf(tenancyPath))
		default:
			// tenant path is explicitly set, we don't change it
		}
	case reflect.Struct:
		_, fv, ok := t.findStructField(v, fieldTenantPath, typeUUIDArr)
		switch {
		case ok:
			fv.Set(reflect.ValueOf(tenancyPath))
		default:
			// tenant path is explicitly set, we don't change it
		}
	default:
		return errors.New("cannot update tenancy automatically, please use struct ptr or map as gorm update target value")
	}
	return nil
}

func (t *Tenancy) findStructField(sv reflect.Value, name string, ft reflect.Type) (f reflect.StructField, fv reflect.Value, ok bool) {
	f, ok = reflectutils.FindStructField(sv.Type(), func(t reflect.StructField) bool {
		return t.Name == name && ft.AssignableTo(t.Type)
	})
	if ok {
		fv = sv.FieldByIndex(f.Index)
	}
	return
}

func (t *Tenancy) findMapValue(mv reflect.Value, keys utils.StringSet, ft reflect.Type) (string, reflect.Value, bool) {
	for iter := mv.MapRange(); iter.Next(); {
		k := iter.Key().String()
		if !keys.Has(k) {
			continue
		}
		v := iter.Value()
		if !v.IsZero() && ft.AssignableTo(v.Type()) {
			return k, v, true
		}
	}
	return "", reflect.Value{}, false
}
