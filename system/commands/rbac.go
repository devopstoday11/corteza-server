package commands

import (
	"fmt"
	cmpsvc "github.com/cortezaproject/corteza-server/compose/service"
	cmptyp "github.com/cortezaproject/corteza-server/compose/types"
	"github.com/cortezaproject/corteza-server/pkg/auth"
	"github.com/cortezaproject/corteza-server/pkg/cli"
	"github.com/cortezaproject/corteza-server/pkg/permissions"
	syssvc "github.com/cortezaproject/corteza-server/system/service"
	systyp "github.com/cortezaproject/corteza-server/system/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"os"
	"sort"
)

// Temporary solution, highly unstable, will change in the future!
type (
	rbacRoleOps map[string][]string

	rbacModule struct {
		res   *cmptyp.Module
		rules permissions.RuleSet

		Allow rbacRoleOps `yaml:"allow"`
		Deny  rbacRoleOps `yaml:"deny"`
	}

	rbacNamespace struct {
		res   *cmptyp.Namespace
		rules permissions.RuleSet

		Allow rbacRoleOps `yaml:"allow"`
		Deny  rbacRoleOps `yaml:"deny"`

		Modules map[string]*rbacModule `yaml:"modules"`
	}

	rbacRoot struct {
		Namespaces map[string]*rbacNamespace `yaml:"namespaces"`
	}

	//rbacRules map[string]permissions.RuleSet

	rbacPreloads struct {
		roles      systyp.RoleSet
		namespaces cmptyp.NamespaceSet
		modules    cmptyp.ModuleSet
	}
)

func RBAC() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rbac",
		Short: "RBAC tools",
		Long:  "Check and manipulates permissions",
	}

	cmd.AddCommand(rbacCheck())

	//cmd.Flags().String("namespace", "", "Import into namespace (by ID or string)")

	return cmd
}

func rbacCheck() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check applied permissions against given file (only supports compose permissions for now)",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				ctx = auth.SetSuperUserContext(cli.Context())
				fh  *os.File
				err error

				r = &rbacRoot{}

				p = rbacPreloads{}

				currentRules = cmpsvc.DefaultPermissions.(interface{ Rules() permissions.RuleSet }).Rules()
			)

			if len(args) > 0 {
				fh, err = os.Open(args[0])
				cli.HandleError(err)
				defer fh.Close()
			} else {
				fh = os.Stdin
			}

			cli.HandleError(yaml.NewDecoder(fh).Decode(r))

			p.roles, _, err = syssvc.DefaultRole.With(ctx).Find(systyp.RoleFilter{})
			cli.HandleError(err)
			p.namespaces, _, err = cmpsvc.DefaultNamespace.With(ctx).Find(cmptyp.NamespaceFilter{})
			cli.HandleError(err)
			p.modules, _, err = cmpsvc.DefaultModule.With(ctx).Find(cmptyp.ModuleFilter{})
			cli.HandleError(err)

			fmt.Printf("Preloaded %d roles(s)\n", len(p.roles))
			fmt.Printf("Preloaded %d namespaces(s)\n", len(p.namespaces))
			fmt.Printf("Preloaded %d module(s)\n", len(p.modules))
			fmt.Printf("Preloaded %d RBAC rule(s)\n", len(currentRules))

			cli.HandleError(r.Resolve(p))

			r.diagnose(currentRules, p)
		},
	}
}

//func (rr rbacRules) Merge(new rbacRules) rbacRules {
//	var out = rr
//
//	for role, rules := range new {
//		if _, has := out[role]; has {
//			out[role] = append(out[role], rules...)
//		} else {
//			out[role] = rules
//		}
//	}
//
//	// @todo implementation
//	return nil
//}

//func (rr rbacRules) Update(resource permissions.Resource, access permissions.Access) {
//	for _, rules := range rr {
//		for _, rule := range rules {
//			rule.Access = access
//		}
//	}
//}

//func (r rbacRoot) CollectRbacRules() rbacRules {
//	rr := rbacRules{}
//
//	for _, ns := range r.Namespaces {
//		rr.Merge(ns.CollectRbacRules())
//	}
//
//	return rr
//}

// Tranverses nodes and resolves references
func (r *rbacRoot) Resolve(p rbacPreloads) (err error) {
	for handle, ns := range r.Namespaces {
		err = ns.Resolve(handle, p)
		if err != nil {
			return
		}
	}

	return nil
}

//func (ns rbacNamespace) CollectRbacRules() rbacRules {
//	var (
//		rr = rbacRules{}
//
//		a = ns.Allow.CollectRbacRules()
//		d = ns.Deny.CollectRbacRules()
//	)
//
//	a.Update(cmptyp.NamespacePermissionResource, permissions.Allow)
//	d.Update(cmptyp.NamespacePermissionResource, permissions.Deny)
//
//	rr = rr.Merge(a).Merge(d)
//
//	for _, m := range ns.Modules {
//		rr = rr.Merge(m.CollectRbacRules())
//	}
//
//	return rr
//}

func (ns *rbacNamespace) Resolve(nsHandle string, p rbacPreloads) error {
	ns.res = p.namespaces.FindByHandle(nsHandle)
	if ns.res == nil {
		return fmt.Errorf("could not find namespace by handle: %q", nsHandle)
	}

	for mHandle, m := range ns.Modules {
		if err := m.Resolve(mHandle, p); err != nil {
			return fmt.Errorf("failed to resolve module on namespace %q: %w", nsHandle, err)
		}
	}

	ns.rules = permissions.RuleSet{}

	if allows, err := ns.Allow.Resolve(ns.res.PermissionResource(), permissions.Allow, p); err != nil {
		return fmt.Errorf("failed to resolve allow rules on namespace %q: %w", nsHandle, err)
	} else {
		ns.rules = append(ns.rules, allows...)
	}

	if allows, err := ns.Deny.Resolve(ns.res.PermissionResource(), permissions.Allow, p); err != nil {
		return fmt.Errorf("failed to resolve deny rules on namespace %q: %w", nsHandle, err)
	} else {
		ns.rules = append(ns.rules, allows...)
	}

	return nil
}

func (ns *rbacNamespace) SortedModuleHandles() []string {
	out := []string{}
	for h := range ns.Modules {
		out = append(out, h)
	}

	sort.Strings(out)
	return out
}

func (m *rbacModule) Resolve(handle string, p rbacPreloads) error {
	var permRes permissions.Resource
	if handle != "*" {
		m.res = p.modules.FindByHandle(handle)
		if m.res == nil {
			return fmt.Errorf("could not find module by handle: %q", handle)
		}

		permRes = m.res.PermissionResource()
	} else {
		permRes = cmptyp.ModulePermissionResource.AppendWildcard()
	}

	m.rules = permissions.RuleSet{}

	if allows, err := m.Allow.Resolve(permRes, permissions.Allow, p); err != nil {
		return fmt.Errorf("failed to resolve allow rules on module %q: %w", handle, err)
	} else {
		m.rules = append(m.rules, allows...)
	}

	if allows, err := m.Deny.Resolve(permRes, permissions.Allow, p); err != nil {
		return fmt.Errorf("failed to resolve deny rules on module %q: %w", handle, err)
	} else {
		m.rules = append(m.rules, allows...)
	}

	return nil
}

func (m *rbacModule) diagnose(currentRules permissions.RuleSet, p rbacPreloads) {
	// all modules
	var (
		res = cmptyp.ModulePermissionResource.AppendWildcard()
	)

	if m.res != nil {
		// specific module
		res = m.res.PermissionResource()
	}

	// all rules that belong to the module
	currentRules = currentRules.ByResource(res)

	printRuleDiffs(currentRules, m.rules, permissions.Allow, p)
	printRuleDiffs(currentRules, m.rules, permissions.Deny, p)

}

func (rules rbacRoleOps) Resolve(res permissions.Resource, access permissions.Access, p rbacPreloads) (permissions.RuleSet, error) {
	prs := permissions.RuleSet{}

	for roleHandle, ops := range rules {
		role := p.roles.FindByHandle(roleHandle)
		if role == nil {
			return nil, fmt.Errorf("could not find role by handle: %q", roleHandle)
		}

		for _, op := range ops {
			prs = append(prs, &permissions.Rule{
				RoleID:    role.ID,
				Resource:  res,
				Operation: permissions.Operation(op),
				Access:    access,
			})
		}
	}

	return prs, nil
}

func (r *rbacRoot) diagnose(c permissions.RuleSet, p rbacPreloads) {
	for _, ns := range r.Namespaces {
		fmt.Printf("=> [%d] %s\n", ns.res.ID, ns.res.Slug)
		fmt.Printf("  checking with %d module(s) from YAML\n", len(ns.Modules))

		if all, has := ns.Modules["*"]; has {
			fmt.Printf("    => ** all modules **\n")
			all.diagnose(c, p)
		}

		for _, handle := range ns.SortedModuleHandles() {
			if handle == "*" {
				continue
			}

			m := ns.Modules[handle]

			if m.res == nil {
				fmt.Printf("    !! \033[33munresolved module with handle %q\033[39m\n", handle)
				continue
			}

			fmt.Printf("    => [%d] %s\n", m.res.ID, m.res.Handle)
			m.diagnose(c, p)
		}
	}
}

func printRuleDiffs(current, required permissions.RuleSet, a permissions.Access, p rbacPreloads) {
	diff := required.ByAccess(a).Diff(current.ByAccess(a))

	if len(diff) > 0 {
		fmt.Printf("       \033[32mmissing %s rules (%d):\033[39m\n", a, len(diff))
		for _, roleID := range diff.Roles() {
			role := p.roles.FindByID(roleID)
			fmt.Printf("        - [%d] %-20s: ", role.ID, role.Handle)
			for _, r := range diff.ByRole(roleID) {
				fmt.Printf(" %s", r.Operation)
			}
			fmt.Println()
		}
	} else {
		fmt.Printf("       \033[32mno missing %s rules\033[39m\n", a)
	}
}
