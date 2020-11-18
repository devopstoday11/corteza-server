package scim

import (
	"context"
	"fmt"
	"github.com/cortezaproject/corteza-server/pkg/auth"
	"github.com/cortezaproject/corteza-server/pkg/errors"
	"github.com/cortezaproject/corteza-server/system/service"
	"github.com/cortezaproject/corteza-server/system/types"
	"github.com/imulab/go-scim/pkg/v2/crud"
	"github.com/imulab/go-scim/pkg/v2/prop"
	"github.com/imulab/go-scim/pkg/v2/spec"
	"net/mail"
	"strconv"
)

type (
	// Vridges corteza user service to SCIM
	bridgeUsers struct {
		users service.UserService
	}
)

// Get a resource by its id. The projection parameter specifies the attributes to be included or excluded from the
// response. Implementations may elect to ignore this parameter in case caller services need all the attributes for
// additional processing.
func (b *bridgeUsers) Get(ctx context.Context, id string, projection *crud.Projection) (*prop.Resource, error) {
	userResType, err := ParseUserResourceType()
	if err != nil {
		return nil, err
	}

	userID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, err
	}

	u, err := b.users.With(ctx).FindByID(userID)
	if err != nil {
		return nil, err
	}

	res := prop.NewResource(userResType)

	return res, res.Navigator().Replace(map[string]interface{}{
		//"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"schemas": []interface{}{
			"urn:ietf:params:scim:schemas:core:2.0:User",
		},
		"id":          id,
		"userName":    u.Username,
		"displayName": u.Name,
		"nickName":    u.Handle,
		"emails": []interface{}{
			map[string]interface{}{
				"value":   u.Email,
				"type":    "work",
				"primary": true,
				"display": u.Email,
			},
		},
	}).Error()
}

// Insert the given resource into the database, or return any error.
func (b *bridgeUsers) Insert(ctx context.Context, resource *prop.Resource) error {
	ctx = auth.SetSuperUserContext(ctx)

	var (
		svc = b.users.With(ctx)
		nav = resource.Navigator()

		u *types.User
	)

	if email, err := extractUserEmail(nav); err != nil {
		return err
	} else if u, err = svc.FindByEmail(email); !errors.Is(err, service.UserErrNotFound()) {
		return err
	} else if errors.Is(err, service.UserErrNotFound()) {
		// no user with such email found; create new
		u = &types.User{Email: email}
	}

	return b.upsert(ctx, u, resource)
}

// Replace overwrites an existing reference resource with the content of the replacement resource. The reference
// and the replacement resource are supposed to have the same id.
func (b *bridgeUsers) Replace(ctx context.Context, ref *prop.Resource, replacement *prop.Resource) error {
	ctx = auth.SetSuperUserContext(ctx)

	var (
		svc = b.users.With(ctx)
		u   *types.User
	)

	if id, err := extractID(ref); err != nil {
		return err
	} else if u, err = svc.FindByID(id); !errors.Is(err, service.UserErrNotFound()) {
		return err
	} else if errors.Is(err, service.UserErrNotFound()) {
		// no user with such email found; create new
		u = &types.User{}
	}

	return b.upsert(ctx, u, replacement)
}

// generic upsert fn
func (b *bridgeUsers) upsert(ctx context.Context, u *types.User, resource *prop.Resource) error {
	ctx = auth.SetSuperUserContext(ctx)

	var (
		svc = b.users.With(ctx)
		err error
		nav = resource.Navigator()
	)

	if v, ok := getStringProp(nav, "displayName"); ok {
		u.Name = v
	}

	if v, ok := getStringProp(nav, "nickName"); ok {
		u.Handle = v
	}

	if v, ok := getStringProp(nav, "userName"); ok {
		u.Username = v
	}

	if u.ID == 0 {
		_, err = svc.Create(u)
	} else {
		_, err = svc.Update(u)
	}

	return err
}

// Count the number of resources that meets the given SCIM filter.
func (b *bridgeUsers) Count(ctx context.Context, filter string) (int, error) {
	return -1, fmt.Errorf("pending implementation")
}

// Delete a resource
func (b *bridgeUsers) Delete(ctx context.Context, resource *prop.Resource) error {
	ctx = auth.SetSuperUserContext(ctx)

	var (
		svc = b.users.With(ctx)
	)

	if id, err := extractID(resource); err != nil {
		return err
	} else {
		return svc.Delete(id)
	}
}

// Query resources. The projection parameter specifies the attributes to be included or excluded from the
// response. Implementations may elect to ignore this parameter in case caller services need all the attributes for
// additional processing.
func (b *bridgeUsers) Query(ctx context.Context, filter string, sort *crud.Sort, pagination *crud.Pagination, projection *crud.Projection) ([]*prop.Resource, error) {
	return nil, fmt.Errorf("pending implementation")
}

func getStringProp(nav prop.Navigator, name string) (string, bool) {
	prop := nav.Dot(name).Current()
	nav.Retract()
	if !prop.IsUnassigned() {
		defer nav.Retract()
		if str, ok := prop.Raw().(string); ok {
			return str, true
		}
	}

	return "", false
}

func extractID(resource *prop.Resource) (uint64, error) {
	var (
		id, _ = strconv.ParseUint(resource.IdOrEmpty(), 10, 64)
	)

	if id == 0 {
		return 0, fmt.Errorf("%w: empty or invalid ID", spec.ErrInvalidValue)
	}

	return id, nil
}

// finds primary (or first email), validates and returns it
func extractUserEmail(nav prop.Navigator) (string, error) {
	var (
		extract = func(d interface{}) (string, error) {
			if d, ok := d.(map[string]interface{}); !ok {
				return "", fmt.Errorf("%w: emails item format invalid", spec.ErrInvalidValue)
			} else if tmp, has := d["value"]; !has {
				return "", fmt.Errorf("%w: email value key missing", spec.ErrInvalidValue)
			} else if email, ok := tmp.(string); !ok {
				return "", fmt.Errorf("%w: email value invalid", spec.ErrInvalidValue)
			} else if tmp, err := mail.ParseAddress(email); err != nil {
				return "", fmt.Errorf("%w: email value %q invalid", spec.ErrInvalidValue, email)
			} else {
				return tmp.Address, nil
			}
		}
	)

	emails, ok := nav.Dot("emails").Current().Raw().([]interface{})
	nav.Retract()
	if !ok {
		return "", fmt.Errorf("%w: emails list format invalid", spec.ErrInvalidValue)
	}

	// find primary email
	for _, email := range emails {
		if email, ok := email.(map[string]interface{}); !ok {
			return "", fmt.Errorf("%w: emails item format invalid", spec.ErrInvalidValue)

		} else if primary, has := email["primary"]; !has {
			continue

		} else if primary, ok := primary.(bool); !ok {
			return "", fmt.Errorf("%w: primary email flag value invalid", spec.ErrInvalidValue)

		} else if primary {
			return extract(email)
		}
	}

	return extract(emails)
}
