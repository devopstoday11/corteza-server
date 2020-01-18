package commands

import (
	"fmt"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/titpetric/factory"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/cortezaproject/corteza-server/pkg/auth"
	"github.com/cortezaproject/corteza-server/pkg/cli"
	"github.com/cortezaproject/corteza-server/pkg/rh"
	"github.com/cortezaproject/corteza-server/system/repository"
	"github.com/cortezaproject/corteza-server/system/service"
	"github.com/cortezaproject/corteza-server/system/types"
)

func Users() *cobra.Command {
	var (
		flagNoPassword bool
	)

	// User management commands.
	cmd := &cobra.Command{
		Use:   "users",
		Short: "User management",
	}

	// List users.
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				ctx = auth.SetSuperUserContext(cli.Context())
				db  = factory.Database.MustGet("system", "default")

				queryFlag = cmd.Flags().Lookup("query").Value.String()
				limitFlag = cmd.Flags().Lookup("limit").Value.String()

				limit int
				err   error
			)

			limit, err = strconv.Atoi(limitFlag)
			cli.HandleError(err)

			userRepo := repository.User(ctx, db)
			uf := types.UserFilter{
				Sort:  "updated_at",
				Query: queryFlag,
				PageFilter: rh.PageFilter{
					PerPage: uint(limit),
				},
			}

			users, _, err := userRepo.Find(uf)
			cli.HandleError(err)

			fmt.Fprintf(
				cmd.OutOrStdout(),
				"                     Created    Updated    EmailAddress",
			)

			for _, u := range users {
				upd := "---- -- --"

				if u.UpdatedAt != nil {
					upd = u.UpdatedAt.Format("2006-01-02")
				}

				fmt.Fprintf(
					cmd.OutOrStdout(),
					"%20d %s %s %-100s %s\n",
					u.ID,
					u.CreatedAt.Format("2006-01-02"),
					upd,
					u.Email,
					u.Name,
				)
			}
		},
	}

	listCmd.Flags().IntP("limit", "l", 20, "How many entry to display")
	listCmd.Flags().StringP("query", "q", "", "Query and filter by handle, email, name")

	addCmd := &cobra.Command{
		Use:   "add [email]",
		Short: "Add new user",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var (
				ctx = auth.SetSuperUserContext(cli.Context())

				db = factory.Database.MustGet("system", "default")

				userRepo = repository.User(ctx, db)
				authSvc  = service.Auth(ctx)

				// @todo email validation
				user = &types.User{Email: args[0]}

				err      error
				password []byte
			)

			if existing, _ := userRepo.FindByEmail(user.Email); existing != nil && existing.ID > 0 {
				cmd.Printf("User already exists [%d].\n", existing.ID)
				return
			}

			if user, err = userRepo.Create(user); err != nil {
				cli.HandleError(err)
			}

			cmd.Printf("User created [%d].\n", user.ID)

			if !flagNoPassword {
				cmd.Print("Set password: ")
				if password, err = terminal.ReadPassword(syscall.Stdin); err != nil {
					cli.HandleError(err)
				}

				if len(password) == 0 {
					// Password not set, that's ok too.
					return
				}

				if err = authSvc.SetPassword(user.ID, string(password)); err != nil {
					cli.HandleError(err)
				}
			}
		},
	}

	addCmd.Flags().BoolVar(
		&flagNoPassword,
		"no-password",
		false,
		"Create user without password")

	pwdCmd := &cobra.Command{
		Use:   "password [email]",
		Short: "Change password for user",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var (
				ctx = auth.SetSuperUserContext(cli.Context())
				db  = factory.Database.MustGet("system", "default")

				userRepo = repository.User(ctx, db)
				authSvc  = service.Auth(ctx)

				user     *types.User
				err      error
				password []byte
			)

			if user, err = userRepo.FindByEmail(args[0]); err != nil {
				cli.HandleError(err)
			}

			cmd.Print("Set password: ")
			if password, err = terminal.ReadPassword(syscall.Stdin); err != nil {
				cli.HandleError(err)
			}

			if len(password) == 0 {
				// Password not set, that's ok too.
				return
			}

			if err = authSvc.SetPassword(user.ID, string(password)); err != nil {
				cli.HandleError(err)
			}
		},
	}

	cmd.AddCommand(
		listCmd,
		addCmd,
		pwdCmd,
	)

	return cmd
}
