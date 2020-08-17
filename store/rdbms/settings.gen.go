package rdbms

// This file is an auto-generated file
//
// Template:    pkg/codegen/assets/store_rdbms.gen.go.tpl
// Definitions: store/settings.yaml
//
// Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated.

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/cortezaproject/corteza-server/store"
	"github.com/cortezaproject/corteza-server/system/types"
	"github.com/jmoiron/sqlx"
)

// SearchSettings returns all matching rows
//
// This function calls convertSettingFilter with the given
// types.SettingsFilter and expects to receive a working squirrel.SelectBuilder
func (s Store) SearchSettings(ctx context.Context, f types.SettingsFilter) (types.SettingValueSet, types.SettingsFilter, error) {
	q, err := s.convertSettingFilter(f)
	if err != nil {
		return nil, f, err
	}

	scap := f.PerPage
	if scap == 0 {
		scap = DefaultSliceCapacity
	}

	if f.Count, err = Count(ctx, s.db, q); err != nil || f.Count == 0 {
		return nil, f, err
	}

	var (
		set = make([]*types.SettingValue, 0, scap)
		// @todo this offset needs to be removed and replaced with key-based-paging
		fetchPage = func(offset, limit uint) (fetched, skipped uint, err error) {
			var (
				res *types.SettingValue
				chk bool
			)

			if limit > 0 {
				q = q.Limit(uint64(limit))
			}

			if offset > 0 {
				q = q.Offset(uint64(offset))
			}

			rows, err := s.Query(ctx, q)
			if err != nil {
				return
			}

			for rows.Next() {
				fetched++
				if res, err = s.internalSettingRowScanner(rows, rows.Err()); err != nil {
					if cerr := rows.Close(); cerr != nil {
						err = fmt.Errorf("could not close rows (%v) after scan error: %w", cerr, err)
					}

					return
				}

				// If check function is set, call it and act accordingly
				if f.Check != nil {
					if chk, err = f.Check(res); err != nil {
						if cerr := rows.Close(); cerr != nil {
							err = fmt.Errorf("could not close rows (%v) after check error: %w", cerr, err)
						}

						return
					} else if !chk {
						// did not pass the check
						// go with the next row
						skipped++
						continue
					}
				}

				set = append(set, res)

				// make sure we do not fetch more than requested!
				if f.Limit > 0 && uint(len(set)) >= f.Limit {
					break
				}
			}

			err = rows.Close()
			return
		}

		fetch = func() error {
			var (
				fetched uint

				// starting offset & limit are from filter arg
				// note that this will have to be improved with key-based pagination
				offset, limit = calculatePaging(f.PageFilter)
			)

			for refetch := 0; refetch < MaxRefetches; refetch++ {
				if fetched, _, err = fetchPage(offset, limit); err != nil {
					return err
				}

				// if limit is not set or we've already collected enough resources
				// we can break the loop right away
				if limit == 0 || fetched == 0 || uint(len(set)) >= f.Limit {
					break
				}

				// we've skipped fetched resources (due to check() fn)
				// and we still have less results (in set) than required by limit
				// inc offset by number of fetched items
				offset += fetched

				if limit < MinRefetchLimit {
					limit = MinRefetchLimit
				}

			}
			return nil
		}
	)

	return set, f, fetch()
}

// LookupSettingByNameOwnedBy searches for settings by name and owner
func (s Store) LookupSettingByNameOwnedBy(ctx context.Context, name string, owned_by uint64) (*types.SettingValue, error) {
	return s.SettingLookup(ctx, squirrel.Eq{
		"st.name":      name,
		"st.rel_owner": owned_by,
	})
}

// CreateSetting creates one or more rows in settings table
func (s Store) CreateSetting(ctx context.Context, rr ...*types.SettingValue) error {
	if len(rr) == 0 {
		return nil
	}

	return Tx(ctx, s.db, s.config, nil, func(db *sqlx.Tx) (err error) {
		for _, res := range rr {
			err = ExecuteSqlizer(ctx, s.DB(), s.Insert(s.SettingTable()).SetMap(s.internalSettingEncoder(res)))
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// UpdateSetting updates one or more existing rows in settings
func (s Store) UpdateSetting(ctx context.Context, rr ...*types.SettingValue) error {
	return s.PartialUpdateSetting(ctx, nil, rr...)
}

// PartialUpdateSetting updates one or more existing rows in settings
//
// It wraps the update into transaction and can perform partial update by providing list of updatable columns
func (s Store) PartialUpdateSetting(ctx context.Context, onlyColumns []string, rr ...*types.SettingValue) error {
	if len(rr) == 0 {
		return nil
	}

	return Tx(ctx, s.db, s.config, nil, func(db *sqlx.Tx) (err error) {
		for _, res := range rr {
			err = s.ExecUpdateSettings(
				ctx,
				squirrel.Eq{s.preprocessColumn("st.name", ""): s.preprocessValue(res.Name, ""),
					s.preprocessColumn("st.rel_owner", ""): s.preprocessValue(res.OwnedBy, ""),
				},
				s.internalSettingEncoder(res).Skip("name", "rel_owner").Only(onlyColumns...))
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// RemoveSetting removes one or more rows from settings table
func (s Store) RemoveSetting(ctx context.Context, rr ...*types.SettingValue) error {
	if len(rr) == 0 {
		return nil
	}

	return Tx(ctx, s.db, s.config, nil, func(db *sqlx.Tx) (err error) {
		for _, res := range rr {
			err = ExecuteSqlizer(ctx, s.DB(), s.Delete(s.SettingTable("st")).Where(squirrel.Eq{s.preprocessColumn("st.name", ""): s.preprocessValue(res.Name, ""),
				s.preprocessColumn("st.rel_owner", ""): s.preprocessValue(res.OwnedBy, ""),
			}))
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// RemoveSettingByNameOwnedBy removes row from the settings table
func (s Store) RemoveSettingByNameOwnedBy(ctx context.Context, name string, ownedBy uint64) error {
	return ExecuteSqlizer(ctx, s.DB(), s.Delete(s.SettingTable("st")).Where(squirrel.Eq{s.preprocessColumn("st.name", ""): s.preprocessValue(name, ""),

		s.preprocessColumn("st.rel_owner", ""): s.preprocessValue(ownedBy, ""),
	}))
}

// TruncateSettings removes all rows from the settings table
func (s Store) TruncateSettings(ctx context.Context) error {
	return Truncate(ctx, s.DB(), s.SettingTable())
}

// ExecUpdateSettings updates all matched (by cnd) rows in settings with given data
func (s Store) ExecUpdateSettings(ctx context.Context, cnd squirrel.Sqlizer, set store.Payload) error {
	return ExecuteSqlizer(ctx, s.DB(), s.Update(s.SettingTable("st")).Where(cnd).SetMap(set))
}

// SettingLookup prepares Setting query and executes it,
// returning types.SettingValue (or error)
func (s Store) SettingLookup(ctx context.Context, cnd squirrel.Sqlizer) (*types.SettingValue, error) {
	return s.internalSettingRowScanner(s.QueryRow(ctx, s.QuerySettings().Where(cnd)))
}

func (s Store) internalSettingRowScanner(row rowScanner, err error) (*types.SettingValue, error) {
	if err != nil {
		return nil, err
	}

	var res = &types.SettingValue{}
	if _, has := s.config.RowScanners["setting"]; has {
		scanner := s.config.RowScanners["setting"].(func(rowScanner, *types.SettingValue) error)
		err = scanner(row, res)
	} else {
		err = row.Scan(
			&res.Name,
			&res.Value,
			&res.OwnedBy,
			&res.UpdatedBy,
			&res.UpdatedAt,
		)
	}

	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("could not scan db row for Setting: %w", err)
	} else {
		return res, nil
	}
}

// QuerySettings returns squirrel.SelectBuilder with set table and all columns
func (s Store) QuerySettings() squirrel.SelectBuilder {
	return s.Select(s.SettingTable("st"), s.SettingColumns("st")...)
}

// SettingTable name of the db table
func (Store) SettingTable(aa ...string) string {
	var alias string
	if len(aa) > 0 {
		alias = " AS " + aa[0]
	}

	return "settings" + alias
}

// SettingColumns returns all defined table columns
//
// With optional string arg, all columns are returned aliased
func (Store) SettingColumns(aa ...string) []string {
	var alias string
	if len(aa) > 0 {
		alias = aa[0] + "."
	}

	return []string{
		alias + "name",
		alias + "value",
		alias + "rel_owner",
		alias + "updated_by",
		alias + "updated_at",
	}
}

// internalSettingEncoder encodes fields from types.SettingValue to store.Payload (map)
//
// Encoding is done by using generic approach or by calling encodeSetting
// func when rdbms.customEncoder=true
func (s Store) internalSettingEncoder(res *types.SettingValue) store.Payload {
	return store.Payload{
		"name":       res.Name,
		"value":      res.Value,
		"rel_owner":  res.OwnedBy,
		"updated_by": res.UpdatedBy,
		"updated_at": res.UpdatedAt,
	}
}
