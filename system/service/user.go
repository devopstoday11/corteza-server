package service

import (
	"context"

	"github.com/pkg/errors"
	"github.com/titpetric/factory"

	"github.com/crusttech/crust/system/repository"
	"github.com/crusttech/crust/system/types"
)

const (
	ErrUserInvalidCredentials = serviceError("UserInvalidCredentials")
	ErrUserLocked             = serviceError("UserLocked")

	uuidLength = 36
)

type (
	user struct {
		db  *factory.DB
		ctx context.Context

		user repository.UserRepository
	}

	UserService interface {
		With(ctx context.Context) UserService

		FindByID(id uint64) (*types.User, error)
		Find(filter *types.UserFilter) (types.UserSet, error)

		Create(input *types.User) (*types.User, error)
		Update(mod *types.User) (*types.User, error)

		FindOrCreate(*types.User) (*types.User, error)
		ValidateCredentials(username, password string) (*types.User, error)
	}
)

func User() UserService {
	return (&user{}).With(context.Background())
}

func (svc *user) With(ctx context.Context) UserService {
	db := repository.DB(ctx)

	return &user{
		db:   db,
		ctx:  ctx,
		user: repository.User(ctx, db),
	}
}

func (svc *user) ValidateCredentials(username, password string) (*types.User, error) {
	user, err := svc.user.FindByUsername(username)
	if err != nil {
		return nil, err
	}

	if !user.ValidatePassword(password) {
		return nil, ErrUserInvalidCredentials
	}

	if !svc.canLogin(user) {
		return nil, ErrUserLocked
	}

	return user, nil
}

func (svc *user) FindByID(id uint64) (*types.User, error) {
	return svc.user.FindByID(id)
}

func (svc *user) Find(filter *types.UserFilter) (types.UserSet, error) {
	return svc.user.Find(filter)
}

// Finds if user with a specific satosa id exists and returns it otherwise it creates a fresh one
func (svc *user) FindOrCreate(user *types.User) (out *types.User, err error) {
	return out, svc.db.Transaction(func() error {
		if len(user.SatosaID) != uuidLength {
			// @todo uuid format check
			return errors.Errorf("Invalid UUID value (%v) for SATOSA ID", user.SatosaID)
		}

		out, err = svc.user.FindBySatosaID(user.SatosaID)

		if err == repository.ErrUserNotFound {
			out, err = svc.user.Create(user)
			return err
		}

		if err != nil {
			// FindBySatosaID error
			return err
		}

		// @todo need to be more selective with fields we update...
		out, err = svc.user.Update(out)
		if err != nil {
			return err
		}

		return nil
	})
}

func (svc *user) Create(input *types.User) (out *types.User, err error) {
	return out, svc.db.Transaction(func() error {
		// Encrypt user password
		if out, err = svc.user.Create(input); err != nil {
			return err
		}
		return nil
	})
}

func (svc *user) Update(mod *types.User) (*types.User, error) {
	return svc.user.Update(mod)
}

func (svc *user) canLogin(u *types.User) bool {
	return u != nil && u.ID > 0 && u.SuspendedAt == nil && u.DeletedAt == nil
}