package mdb

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"strconv"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type CreateClause struct {
	Field *schema.Field
}

func (v CreateClause) Build(clause.Builder) {
}

func (v CreateClause) MergeClause(*clause.Clause) {
}

func (v CreateClause) ModifyStatement(stmt *gorm.Statement) {
	switch stmt.ReflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < stmt.ReflectValue.Len(); i++ {
			v.setVersionColumn(stmt, stmt.ReflectValue.Index(i))
		}
	case reflect.Struct:
		v.setVersionColumn(stmt, stmt.ReflectValue)
	}
}

func (v CreateClause) Name() string {
	return ""
}

func (v CreateClause) setVersionColumn(stmt *gorm.Statement, reflectValue reflect.Value) {
	var value int64 = 1
	if val, zero := v.Field.ValueOf(stmt.Context, reflectValue); !zero {
		_v, ok := isVersionValue(val)
		if ok {
			value = _v
		}
	}
	_ = v.Field.Set(stmt.Context, reflectValue, value)
}

type UpdateClause struct {
	Field *schema.Field
}

func (v UpdateClause) Build(clause.Builder) {
}

func (v UpdateClause) MergeClause(*clause.Clause) {
}

func (v UpdateClause) ModifyStatement(stmt *gorm.Statement) {
	if _, ok := stmt.Clauses["version_enabled"]; ok {
		return
	}

	if c, ok := stmt.Clauses["WHERE"]; ok {
		if where, ok := c.Expression.(clause.Where); ok && len(where.Exprs) > 1 {
			for _, expr := range where.Exprs {
				if orCond, ok := expr.(clause.OrConditions); ok && len(orCond.Exprs) == 1 {
					where.Exprs = []clause.Expression{clause.And(where.Exprs...)}
					c.Expression = where
					stmt.Clauses["WHERE"] = c
					break
				}
			}
		}
	}

	if !stmt.Unscoped {
		if val, zero := v.Field.ValueOf(stmt.Context, stmt.ReflectValue); !zero {
			_v, ok := isVersionValue(val)
			if ok {
				stmt.AddClause(clause.Where{Exprs: []clause.Expression{
					clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: v.Field.DBName}, Value: _v},
				}})
			}
		}
	}

	// struct to map[string]interface{}. version field is int64, but needs to set its value to string
	dv := reflect.ValueOf(stmt.Dest)
	if reflect.Indirect(dv).Kind() == reflect.Struct {
		selectColumns, restricted := stmt.SelectAndOmitColumns(false, true)

		sd, _ := schema.Parse(stmt.Dest, &sync.Map{}, stmt.DB.NamingStrategy)
		d := make(map[string]interface{})
		for _, field := range sd.Fields {
			if field.DBName == v.Field.DBName {
				continue
			}
			if field.DBName == "" {
				continue
			}

			if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && (!restricted || !stmt.SkipHooks)) {
				if field.AutoUpdateTime > 0 {
					continue
				}

				val, isZero := field.ValueOf(stmt.Context, dv)
				if (ok || !isZero) && field.Updatable {
					d[field.DBName] = val
				}
			}
		}

		stmt.Dest = d
	}

	stmt.SetColumn(v.Field.DBName, clause.Expr{SQL: stmt.Quote(v.Field.DBName) + "+1"}, true)
	stmt.Clauses["version_enabled"] = clause.Clause{}
}

func (v UpdateClause) Name() string {
	return ""
}

type Version sql.NullInt64

func (v *Version) CreateClauses(field *schema.Field) []clause.Interface {
	return []clause.Interface{CreateClause{Field: field}}
}

func (v *Version) MarshalJSON() ([]byte, error) {
	if v.Valid {
		return strconv.AppendInt(nil, v.Int64, 10), nil
	}
	return []byte("null"), nil
}

func (v *Version) Scan(value interface{}) error {
	return (*sql.NullInt64)(v).Scan(value)
}

func (v *Version) UnmarshalJSON(bytes []byte) error {
	if string(bytes) == "null" {
		v.Valid = false
		return nil
	}
	err := json.Unmarshal(bytes, &v.Int64)
	if err == nil {
		v.Valid = true
	}
	return err
}

func (v *Version) UpdateClauses(field *schema.Field) []clause.Interface {
	return []clause.Interface{UpdateClause{Field: field}}
}

func (v *Version) Value() (driver.Value, error) {
	if !v.Valid {
		return nil, nil
	}
	return v.Int64, nil
}

func isVersionValue(val interface{}) (int64, bool) {
	switch version := val.(type) {
	case Version:
		return version.Int64, true
	case *Version:
		return version.Int64, true
	}
	return 1, false
}
