package scim

import (
	gojson "encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/imulab/go-scim/pkg/v2/handlerutil"
	"github.com/imulab/go-scim/pkg/v2/json"
	"github.com/imulab/go-scim/pkg/v2/service"
	"github.com/imulab/go-scim/pkg/v2/spec"
	"go.uber.org/zap"
	"net/http"
)

func idParam(w http.ResponseWriter, r *http.Request, log *zap.Logger, id *string) bool {
	*id = chi.URLParam(r, "id")
	if len(*id) == 0 {
		err := fmt.Errorf("%w: id is empty", spec.ErrInvalidSyntax)
		log.Error("error receiving get request", zap.Error(err))
		_ = handlerutil.WriteError(w, err)
		return false
	}

	return true
}

// GetHandler returns a route handler function for getting SCIM resource.
func GetHandler(svc service.Get, log *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var id string
		if !idParam(w, r, log, &id) {
			return
		}

		projection, err := handlerutil.GetRequestProjection(r)
		if err != nil {
			log.Error("error parsing getting request", zap.Error(err))
			_ = handlerutil.WriteError(w, err)
			return
		}

		resp, err := svc.Do(r.Context(), &service.GetRequest{
			ResourceID: id,
			Projection: projection,
		})
		if err != nil {
			log.Error("error when getting resource", zap.Error(err))
			_ = handlerutil.WriteError(w, err)
			return
		}

		var opt []json.Options
		if projection != nil {
			if len(projection.Attributes) > 0 {
				opt = append(opt, json.Include(projection.Attributes...))
			}
			if len(projection.ExcludedAttributes) > 0 {
				opt = append(opt, json.Exclude(projection.ExcludedAttributes...))
			}
		}

		_ = handlerutil.WriteResourceToResponse(w, resp.Resource, opt...)
	}
}

// CreateHandler returns a route handler function for creating SCIM resources.
func CreateHandler(svc service.Create, log *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cr, closer := handlerutil.CreateRequest(r)
		defer closer()

		resp, err := svc.Do(r.Context(), cr)
		if err != nil {
			log.Error("error when creating resource", zap.Error(err))
			_ = handlerutil.WriteError(w, err)
			return
		}

		w.WriteHeader(http.StatusCreated)
		_ = handlerutil.WriteResourceToResponse(w, resp.Resource)
	}
}

// DeleteHandler returns a route handler function for deleting SCIM resource.
func DeleteHandler(svc service.Delete, log *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var id string
		if !idParam(w, r, log, &id) {
			return
		}

		_, err := svc.Do(r.Context(), handlerutil.DeleteRequest(r)(id))
		if err != nil {
			log.Error("error when deleting resource", zap.Error(err))
			_ = handlerutil.WriteError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ReplaceHandler returns a route handler function for replacing SCIM resource.
func ReplaceHandler(svc service.Replace, log *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var id string
		if !idParam(w, r, log, &id) {
			return
		}

		reqFunc, closer := handlerutil.ReplaceRequest(r)
		defer closer()

		resp, err := svc.Do(r.Context(), reqFunc(id))
		if err != nil {
			log.Error("error when replacing resource", zap.Error(err))
			_ = handlerutil.WriteError(w, err)
			return
		}

		if !resp.Replaced {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		_ = handlerutil.WriteResourceToResponse(w, resp.Resource)
	}
}

// ServiceProviderConfigHandler returns a http route handler to write service provider config info.
func ServiceProviderConfigHandler(config *spec.ServiceProviderConfig) func(w http.ResponseWriter, r *http.Request) {
	raw, err := gojson.Marshal(config)
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", spec.ApplicationScimJson)
		_, _ = w.Write(raw)
	}
}
