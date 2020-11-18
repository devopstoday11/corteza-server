package scim

import (
	"encoding/json"
	"github.com/cortezaproject/corteza-server/pkg/logger"
	"github.com/cortezaproject/corteza-server/pkg/options"
	"github.com/cortezaproject/corteza-server/system/scim/assets"
	"github.com/cortezaproject/corteza-server/system/service"
	"github.com/go-chi/chi"
	"github.com/goware/statik/fs"
	"github.com/imulab/go-scim/pkg/v2/crud"
	scimService "github.com/imulab/go-scim/pkg/v2/service"
	"github.com/imulab/go-scim/pkg/v2/spec"
	"go.uber.org/zap"
	"net/http"
)

var (
	embedded http.FileSystem
)

func init() {
	var err error
	embedded, err = fs.New(assets.Asset)
	if err != nil {
		panic(err)
	}
}

func Guard(opt options.SCIMOpt) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// temp authorization mechanism so we do not have to
		// pre-create users and generate their auth tokens
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authPrefix := "Bearer "
			authHeader := r.Header.Get("Authorization")
			if (len(authPrefix)+len(opt.Secret)) == len(authHeader) && opt.Secret == authHeader[len(authPrefix):] {
				// all good, auth header matches the secret
				next.ServeHTTP(w, r)
				return
			}

			http.Error(w, "Unauthorized", http.StatusForbidden)

		})
	}
}

func Routes(r chi.Router) {
	var (
		log = logger.Default().Named("scim").WithOptions(zap.AddStacktrace(zap.PanicLevel))
		db  = &bridgeUsers{users: service.DefaultUser}

		cfg = &spec.ServiceProviderConfig{
			Schemas: nil,
			DocURI:  "",
			Patch: struct {
				Supported bool `json:"supported"`
			}{false},
			Bulk: struct {
				Supported  bool `json:"supported"`
				MaxOp      int  `json:"maxOperations"`
				MaxPayload int  `json:"maxPayloadSize"`
			}{false, 0, 0},
			Filter: struct {
				Supported  bool `json:"supported"`
				MaxResults int  `json:"maxResults"`
			}{false, 0},
			ChangePassword: struct {
				Supported bool `json:"supported"`
			}{false},
			Sort: struct {
				Supported bool `json:"supported"`
			}{false},
			ETag: struct {
				Supported bool `json:"supported"`
			}{false},
			AuthSchemes: nil,
		}
	)

	err := registerSchemas(
		"/schemas/core_schema.json",
		"/schemas/user_schema.json",
		"/schemas/user_enterprise_extension_schema.json",
		"/schemas/group_schema.json",
	)

	if err != nil {
		log.Error("failed to register schemas", zap.Error(err))
		return
	}

	userResType, err := ParseUserResourceType()
	if err != nil {
		log.Error("failed to register schemas", zap.Error(err))
		return
	}

	r.Get("/ServiceProviderConfig", ServiceProviderConfigHandler(cfg))
	//r.Get("/Schemas", SchemasHandler())
	//r.Get("/Schemas/{id}", SchemaByIdHandler())
	//r.Get("/ResourceTypes", ResourceTypesHandler(app.UserResourceType(), app.GroupResourceType()))
	//r.Get("/ResourceTypes/{id}", ResourceTypeByIdHandler(app.userResourceType, app.GroupResourceType()))

	r.Route("/Users", func(r chi.Router) {
		r.Get("/{id}", GetHandler(scimService.GetService(db), log))
		r.Post("/", CreateHandler(scimService.CreateService(userResType, db, nil), log))
		r.Put("/{id}", ReplaceHandler(scimService.ReplaceService(cfg, userResType, db, nil), log))
		//r.Patch("/{id}", PatchHandler(scimService.PatchService(cfg, db, nil, nil), log))
		r.Delete("/{id}", DeleteHandler(scimService.DeleteService(cfg, db), log))
	})

	//r.Get("/Groups/{id}", GetHandler(app.GroupGetService(), log))
	//r.Get("/Groups", SearchHandler(app.GroupQueryService(), log))
	//r.Post("/Groups", CreateHandler(app.GroupCreateService(), log))
	//r.Put("/Groups/{id}", ReplaceHandler(app.GroupReplaceService(), log))
	//r.Patch("/Groups/{id}", PatchHandler(app.GroupPatchService(), log))
	//r.Delete("/Groups/{id}", DeleteHandler(app.GroupDeleteService(), log))
}

// ParseUserResourceType returns the parsed spec.ResourceType from the JSON schema definition at UserResourceTypePath.
// Caller must make sure RegisterSchemas was invoked first.
func ParseUserResourceType() (*spec.ResourceType, error) {
	t, err := parseResourceType("/resource_types/user_resource_type.json")
	if err != nil {
		return nil, err
	}
	crud.Register(t)
	return t, nil

}

func registerSchemas(pp ...string) error {
	for _, path := range pp {
		var (
			schema = new(spec.Schema)
		)

		if f, err := embedded.Open(path); err != nil {
			return err
		} else if err = json.NewDecoder(f).Decode(schema); err != nil {
			return err
		} else {
			spec.Schemas().Register(schema)
		}
	}

	return nil
}

func parseResourceType(path string) (*spec.ResourceType, error) {
	f, err := embedded.Open(path)
	if err != nil {
		return nil, err
	}

	rt := new(spec.ResourceType)
	err = json.NewDecoder(f).Decode(rt)
	if err != nil {
		return nil, err
	}

	return rt, nil
}
