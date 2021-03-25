package migration

import (
	"context"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/utils"
	"gorm.io/gorm"
)

type MigrationFunc func(ctx context.Context) error

type Registrar struct {
	migrationSteps []*Migration
	versions 	   utils.StringSet
	errs 		   []error
}

func (r *Registrar) AddMigrations(m... *Migration) {
	r.migrationSteps = append(r.migrationSteps, m...)
}

type Migration struct {
	Version     Version
	Description string
	Func		MigrationFunc
	Tags        utils.StringSet
}

/*	v, err := fromString(version)

	if err != nil {
		r.errs = append(r.errs, err)
		return
	}

	if r.versions.Has(version) {
		r.errs = append(r.errs, errors.New(fmt.Sprintf("a migration step with version %s already exist", version)))
		return
	}

	m := &Migration{
		Version:     v,
		Description: description,
		Func:        migrationFunc,
		Tags: utils.NewStringSet(tags...),
	}
 */

func WithVersion(version string) *Migration {
	v, err := fromString(version)
	if err != nil {
		panic(err)
	}
	return &Migration{
		Version: v,
	}
}

func (m *Migration) Dot(i int) *Migration {
	m.Version = append(m.Version, i)
	return m
}

func (m *Migration) WithTag(tags...string) *Migration {
	if m.Tags == nil {
		m.Tags = utils.NewStringSet(tags...)
	} else {
		m.Tags.Add(tags...)
	}
	return m
}

func (m *Migration) WithFile(filePath string, db *gorm.DB) *Migration {
	m.Func = migrationFuncFromTextFile(filePath, db)
	return m
}

func (m *Migration) WithFunc(f MigrationFunc) *Migration {
	m.Func = f
	return m
}

func (m *Migration) WithDesc(d string) *Migration {
	m.Description = d
	return m
}