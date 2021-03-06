package types

import (
	"time"
	"unicode/utf8"

	"github.com/cortezaproject/corteza-server/compose/types"
)

const (
	// SfDateTimeLayout represents the date-time template used by sales force
	SfDateTimeLayout = time.RFC3339
	// DateOnlyLayout represents our internal date only date-time fields
	DateOnlyLayout = "2006-01-02"
	// TimeOnlyLayout represents our internal time only date-time fields
	TimeOnlyLayout = "15:04:05Z"

	// EvalPrefix defines the prefix used by formulas, defined by value mapping
	EvalPrefix = "=EVL="

	UserModHandle = "User"

	MetaMapExt   = ".map.json"
	MetaJoinExt  = ".join.json"
	MetaValueExt = ".value.json"

	LegacyRefIDField = "sys_legacy_ref_id"
)

var (
	// ExprLang contains gval language that should be used for any expression evaluation
	ExprLang = GLang()

	ModulesGlobal = make(types.ModuleSet, 0)
)

type (
	// PostProc is used withing channels, used by the import process.
	PostProc struct {
		// Leafs defines a set of leaf nodes that can be imported next.
		Leafs []*ImportNode
		// ...
		Err error
		// Node contains the current node; usefull for debugging
		Node *ImportNode
	}
)

// helper function for removing invalid UTF runes from the given string.
// SalesForce &nbsp; with a special character, that is not supported in our char set
func fixUtf(r rune) rune {
	if r == utf8.RuneError {
		return -1
	}
	return r
}
