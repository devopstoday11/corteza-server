package rdbms

// This file is an auto-generated file
//
// Template:    pkg/codegen/assets/store_rdbms.gen.go.tpl
// Definitions: store/rbac_rules.yaml
//
// Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated.

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/cortezaproject/corteza-server/pkg/permissions"
	"github.com/cortezaproject/corteza-server/store"
)

// SearchRbacRules returns all matching rows
//
// This function calls convertRbacRuleFilter with the given
// permissions.RuleFilter and expects to receive a working squirrel.SelectBuilder
func (s Store) SearchRbacRules(ctx context.Context, f permissions.RuleFilter) (permissions.RuleSet, permissions.RuleFilter, error) {
	var scap uint
	q := s.rbacRulesSelectBuilder()

	if scap == 0 {
		scap = DefaultSliceCapacity
	}

	var (
		set = make([]*permissions.Rule, 0, scap)
		// Paging is disabled in definition yaml file
		// {search: {enablePaging:false}} and this allows
		// a much simpler row fetching logic
		fetch = func() error {
			var (
				res       *permissions.Rule
				rows, err = s.Query(ctx, q)
			)

			if err != nil {
				return err
			}

			for rows.Next() {
				if rows.Err() == nil {
					res, err = s.internalRbacRuleRowScanner(rows)
				}

				if err != nil {
					if cerr := rows.Close(); cerr != nil {
						err = fmt.Errorf("could not close rows (%v) after scan error: %w", cerr, err)
					}

					return err
				}

				// If check function is set, call it and act accordingly
				set = append(set, res)
			}

			return rows.Close()
		}
	)

	return set, f, s.config.ErrorHandler(fetch())
}

// CreateRbacRule creates one or more rows in rbac_rules table
func (s Store) CreateRbacRule(ctx context.Context, rr ...*permissions.Rule) (err error) {
	for _, res := range rr {
		err = s.execCreateRbacRules(ctx, s.internalRbacRuleEncoder(res))
		if err != nil {
			return err
		}
	}

	return
}

// UpdateRbacRule updates one or more existing rows in rbac_rules
func (s Store) UpdateRbacRule(ctx context.Context, rr ...*permissions.Rule) error {
	return s.config.ErrorHandler(s.PartialRbacRuleUpdate(ctx, nil, rr...))
}

// PartialRbacRuleUpdate updates one or more existing rows in rbac_rules
func (s Store) PartialRbacRuleUpdate(ctx context.Context, onlyColumns []string, rr ...*permissions.Rule) (err error) {
	for _, res := range rr {
		err = s.execUpdateRbacRules(
			ctx,
			squirrel.Eq{
				s.preprocessColumn("rls.rel_role", ""):  s.preprocessValue(res.RoleID, ""),
				s.preprocessColumn("rls.resource", ""):  s.preprocessValue(res.Resource, ""),
				s.preprocessColumn("rls.operation", ""): s.preprocessValue(res.Operation, ""),
			},
			s.internalRbacRuleEncoder(res).Skip("rel_role", "resource", "operation").Only(onlyColumns...))
		if err != nil {
			return s.config.ErrorHandler(err)
		}
	}

	return
}

// UpsertRbacRule updates one or more existing rows in rbac_rules
func (s Store) UpsertRbacRule(ctx context.Context, rr ...*permissions.Rule) (err error) {
	for _, res := range rr {
		err = s.config.ErrorHandler(s.execUpsertRbacRules(ctx, s.internalRbacRuleEncoder(res)))
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteRbacRule Deletes one or more rows from rbac_rules table
func (s Store) DeleteRbacRule(ctx context.Context, rr ...*permissions.Rule) (err error) {
	for _, res := range rr {
		err = s.execDeleteRbacRules(ctx, squirrel.Eq{
			s.preprocessColumn("rls.rel_role", ""):  s.preprocessValue(res.RoleID, ""),
			s.preprocessColumn("rls.resource", ""):  s.preprocessValue(res.Resource, ""),
			s.preprocessColumn("rls.operation", ""): s.preprocessValue(res.Operation, ""),
		})
		if err != nil {
			return s.config.ErrorHandler(err)
		}
	}

	return nil
}

// DeleteRbacRuleByRoleIDResourceOperation Deletes row from the rbac_rules table
func (s Store) DeleteRbacRuleByRoleIDResourceOperation(ctx context.Context, roleID uint64, resource string, operation string) error {
	return s.execDeleteRbacRules(ctx, squirrel.Eq{
		s.preprocessColumn("rls.rel_role", ""): s.preprocessValue(roleID, ""),

		s.preprocessColumn("rls.resource", ""): s.preprocessValue(resource, ""),

		s.preprocessColumn("rls.operation", ""): s.preprocessValue(operation, ""),
	})
}

// TruncateRbacRules Deletes all rows from the rbac_rules table
func (s Store) TruncateRbacRules(ctx context.Context) error {
	return s.config.ErrorHandler(s.Truncate(ctx, s.rbacRuleTable()))
}

// execLookupRbacRule prepares RbacRule query and executes it,
// returning permissions.Rule (or error)
func (s Store) execLookupRbacRule(ctx context.Context, cnd squirrel.Sqlizer) (res *permissions.Rule, err error) {
	var (
		row rowScanner
	)

	row, err = s.QueryRow(ctx, s.rbacRulesSelectBuilder().Where(cnd))
	if err != nil {
		return
	}

	res, err = s.internalRbacRuleRowScanner(row)
	if err != nil {
		return
	}

	return res, nil
}

// execCreateRbacRules updates all matched (by cnd) rows in rbac_rules with given data
func (s Store) execCreateRbacRules(ctx context.Context, payload store.Payload) error {
	return s.config.ErrorHandler(s.Exec(ctx, s.InsertBuilder(s.rbacRuleTable()).SetMap(payload)))
}

// execUpdateRbacRules updates all matched (by cnd) rows in rbac_rules with given data
func (s Store) execUpdateRbacRules(ctx context.Context, cnd squirrel.Sqlizer, set store.Payload) error {
	return s.config.ErrorHandler(s.Exec(ctx, s.UpdateBuilder(s.rbacRuleTable("rls")).Where(cnd).SetMap(set)))
}

// execUpsertRbacRules inserts new or updates matching (by-primary-key) rows in rbac_rules with given data
func (s Store) execUpsertRbacRules(ctx context.Context, set store.Payload) error {
	upsert, err := s.config.UpsertBuilder(
		s.config,
		s.rbacRuleTable(),
		set,
		"rel_role",
		"resource",
		"operation",
	)

	if err != nil {
		return err
	}

	return s.config.ErrorHandler(s.Exec(ctx, upsert))
}

// execDeleteRbacRules Deletes all matched (by cnd) rows in rbac_rules with given data
func (s Store) execDeleteRbacRules(ctx context.Context, cnd squirrel.Sqlizer) error {
	return s.config.ErrorHandler(s.Exec(ctx, s.DeleteBuilder(s.rbacRuleTable("rls")).Where(cnd)))
}

func (s Store) internalRbacRuleRowScanner(row rowScanner) (res *permissions.Rule, err error) {
	res = &permissions.Rule{}

	if _, has := s.config.RowScanners["rbacRule"]; has {
		scanner := s.config.RowScanners["rbacRule"].(func(_ rowScanner, _ *permissions.Rule) error)
		err = scanner(row, res)
	} else {
		err = row.Scan(
			&res.RoleID,
			&res.Resource,
			&res.Operation,
			&res.Access,
		)
	}

	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("could not scan db row for RbacRule: %w", err)
	} else {
		return res, nil
	}
}

// QueryRbacRules returns squirrel.SelectBuilder with set table and all columns
func (s Store) rbacRulesSelectBuilder() squirrel.SelectBuilder {
	return s.SelectBuilder(s.rbacRuleTable("rls"), s.rbacRuleColumns("rls")...)
}

// rbacRuleTable name of the db table
func (Store) rbacRuleTable(aa ...string) string {
	var alias string
	if len(aa) > 0 {
		alias = " AS " + aa[0]
	}

	return "rbac_rules" + alias
}

// RbacRuleColumns returns all defined table columns
//
// With optional string arg, all columns are returned aliased
func (Store) rbacRuleColumns(aa ...string) []string {
	var alias string
	if len(aa) > 0 {
		alias = aa[0] + "."
	}

	return []string{
		alias + "rel_role",
		alias + "resource",
		alias + "operation",
		alias + "access",
	}
}

// {true true false false false}

// internalRbacRuleEncoder encodes fields from permissions.Rule to store.Payload (map)
//
// Encoding is done by using generic approach or by calling encodeRbacRule
// func when rdbms.customEncoder=true
func (s Store) internalRbacRuleEncoder(res *permissions.Rule) store.Payload {
	return store.Payload{
		"rel_role":  res.RoleID,
		"resource":  res.Resource,
		"operation": res.Operation,
		"access":    res.Access,
	}
}
