package rh

import (
	"github.com/Masterminds/squirrel"
	"github.com/lann/builder"
	"github.com/pkg/errors"
	"github.com/titpetric/factory"
)

// FetchOne fetches single row from table
func FetchOne(db *factory.DB, q squirrel.SelectBuilder, one interface{}) (err error) {
	var (
		sql  string
		args []interface{}
	)

	if sql, args, err = q.ToSql(); err != nil {
		return
	}

	if err = db.Get(one, sql, args...); err != nil {
		return
	}

	return
}

// Count counts all rows that match conditions from given query builder
func Count(db *factory.DB, q squirrel.SelectBuilder) (count uint, err error) {
	// Remove order-bys for counting
	q = builder.Delete(q, "OrderBys").(squirrel.SelectBuilder)

	// Replace columns
	q = builder.Delete(q, "Columns").(squirrel.SelectBuilder).Column("COUNT(*)")

	if sqlSelect, argsSelect, err := q.ToSql(); err != nil {
		return 0, err
	} else {
		if err := db.Get(&count, sqlSelect, argsSelect...); err != nil {
			return 0, err
		}
	}

	return count, nil
}

// FetchPaged fetches paged rows
func FetchPaged(db *factory.DB, q squirrel.SelectBuilder, p PageFilter, set interface{}) error {
	if p.Limit+p.Offset == 0 {
		// When both, offset & limit are 0,
		// calculate both values from page/perPage params
		if p.PerPage > 0 {
			p.Limit = p.PerPage
		}

		if p.Page < 1 {
			p.Page = 1
		}

		p.Offset = uint((p.Page - 1) * p.PerPage)
	}

	if p.Limit > 0 {
		q = q.Limit(uint64(p.Limit))
	}

	if p.Offset > 0 {
		q = q.Offset(uint64(p.Offset))
	}

	return FetchAll(db, q, set)
}

// FetchPaged fetches paged rows
func FetchAll(db *factory.DB, q squirrel.Sqlizer, set interface{}) error {
	if sqlSelect, argsSelect, err := q.ToSql(); err != nil {
		return err
	} else {
		return db.Select(set, sqlSelect, argsSelect...)
	}
}

// IsFound helps with one-row results
func IsFound(err error, valid bool, nerr error) error {
	if err != nil {
		return errors.WithStack(err)
	} else if !valid {
		return errors.WithStack(nerr)
	}

	return nil
}
