// Package testoutput provides primitives to interact with the openapi HTTP API.
//
// Code generated by unknown module path version unknown version DO NOT EDIT.
package testoutput

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime"
	strictecho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// ProfileProfile defines model for ProfileProfile.
type ProfileProfile struct {
	Dob      ProfileZeroableTime   `json:"dob,omitempty"`
	Email    ProfileZeroableString `json:"email,omitempty"`
	Id       ProfileUUID           `json:"id,omitempty"`
	Name     ProfileZeroableString `json:"name,omitempty"`
	Nin      ProfileZeroableString `json:"nin,omitempty"`
	Phone    ProfileZeroableString `json:"phone,omitempty"`
	TenantId ProfileUUID           `json:"tenant_id,omitempty"`
}

// ProfileUUID defines model for ProfileUUID.
type ProfileUUID = openapi_types.UUID

// ProfileZeroableString defines model for ProfileZeroableString.
type ProfileZeroableString = string

// ProfileZeroableTime defines model for ProfileZeroableTime.
type ProfileZeroableTime = time.Time

// ZeroableBoolean defines model for ZeroableBoolean.
type ZeroableBoolean = bool

// ProfileProfileID defines model for ProfileProfileID.
type ProfileProfileID = ProfileUUID

// ProfileError defines model for ProfileError.
type ProfileError struct {
	Id ProfileUUID `json:"id,omitempty"`
}

// GetProfileParams defines parameters for GetProfile.
type GetProfileParams struct {
	Validate  *ZeroableBoolean `form:"validate,omitempty" json:"validate,omitempty"`
	SomeQuery *string          `form:"some-query,omitempty" json:"some-query,omitempty"`
}

// PutProfileParams defines parameters for PutProfile.
type PutProfileParams struct {
	SomeQuery *string `form:"some-query,omitempty" json:"some-query,omitempty"`
}

// GetValidatedProfileParams defines parameters for GetValidatedProfile.
type GetValidatedProfileParams struct {
	SomeQuery *string `form:"some-query,omitempty" json:"some-query,omitempty"`
}

/* ignored */

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// get profile
	// (GET /profiles/{profile-id})
	GetProfile(ctx echo.Context, profileId ProfileProfileID, params GetProfileParams) error
	// Create/Update profile
	// (PUT /profiles/{profile-id})
	PutProfile(ctx echo.Context, profileId ProfileProfileID, params PutProfileParams) error
	// get profile
	// (GET /validated-profiles/{profile-id})
	GetValidatedProfile(ctx echo.Context, profileId ProfileProfileID, params GetValidatedProfileParams) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// GetProfile converts echo context to params.
func (w *ServerInterfaceWrapper) GetProfile(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "profile-id" -------------
	var profileId ProfileProfileID

	err = runtime.BindStyledParameterWithOptions("simple", "profile-id", ctx.Param("profile-id"), &profileId, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter profile-id: %s", err))
	}

	// Parameter object where we will unmarshal all parameters from the context
	var params GetProfileParams
	// ------------- Optional query parameter "validate" -------------

	err = runtime.BindQueryParameter("form", true, false, "validate", ctx.QueryParams(), &params.Validate)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter validate: %s", err))
	}

	// ------------- Optional query parameter "some-query" -------------

	err = runtime.BindQueryParameter("form", true, false, "some-query", ctx.QueryParams(), &params.SomeQuery)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter some-query: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.GetProfile(ctx, profileId, params)
	return err
}

// PutProfile converts echo context to params.
func (w *ServerInterfaceWrapper) PutProfile(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "profile-id" -------------
	var profileId ProfileProfileID

	err = runtime.BindStyledParameterWithOptions("simple", "profile-id", ctx.Param("profile-id"), &profileId, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter profile-id: %s", err))
	}

	// Parameter object where we will unmarshal all parameters from the context
	var params PutProfileParams
	// ------------- Optional query parameter "some-query" -------------

	err = runtime.BindQueryParameter("form", true, false, "some-query", ctx.QueryParams(), &params.SomeQuery)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter some-query: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.PutProfile(ctx, profileId, params)
	return err
}

// GetValidatedProfile converts echo context to params.
func (w *ServerInterfaceWrapper) GetValidatedProfile(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "profile-id" -------------
	var profileId ProfileProfileID

	err = runtime.BindStyledParameterWithOptions("simple", "profile-id", ctx.Param("profile-id"), &profileId, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter profile-id: %s", err))
	}

	// Parameter object where we will unmarshal all parameters from the context
	var params GetValidatedProfileParams
	// ------------- Optional query parameter "some-query" -------------

	err = runtime.BindQueryParameter("form", true, false, "some-query", ctx.QueryParams(), &params.SomeQuery)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter some-query: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.GetValidatedProfile(ctx, profileId, params)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {
	RegisterHandlersWithBaseURL(router, si, "")
}

// Registers handlers, and prepends BaseURL to the paths, so that the paths
// can be served under a prefix.
func RegisterHandlersWithBaseURL(router EchoRouter, si ServerInterface, baseURL string) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.GET(baseURL+"/profiles/:profile-id", wrapper.GetProfile)
	router.PUT(baseURL+"/profiles/:profile-id", wrapper.PutProfile)
	router.GET(baseURL+"/validated-profiles/:profile-id", wrapper.GetValidatedProfile)

}

/* ignored */
type strictRequest interface {
	ToRequest(base *http.Request) (*http.Request, error)
}

type GetProfileRequestObject struct {
	ProfileId ProfileProfileID `json:"profile-id"`
	Params    GetProfileParams
}

func (r GetProfileRequestObject) ToRequest(base *http.Request) (*http.Request, error) {
	return base, nil
}

type PutProfileRequestObject struct {
	ProfileId ProfileProfileID `json:"profile-id"`
	Params    PutProfileParams
}

func (r PutProfileRequestObject) ToRequest(base *http.Request) (*http.Request, error) {
	return base, nil
}

type GetValidatedProfileRequestObject struct {
	ProfileId ProfileProfileID `json:"profile-id"`
	Params    GetValidatedProfileParams
}

func (r GetValidatedProfileRequestObject) ToRequest(base *http.Request) (*http.Request, error) {
	return base, nil
}

// StrictServerInterface represents all server handlers.
type StrictServerInterface interface {
	// get profile
	// (GET /profiles/{profile-id})
	GetProfile(ctx context.Context, request GetProfileRequestObject) (UpstreamProfileGetProfileRequestObject, error)
	// Create/Update profile
	// (PUT /profiles/{profile-id})
	PutProfile(ctx context.Context, request PutProfileRequestObject) (UpstreamProfilePutProfileRequestObject, error)
	// get profile
	// (GET /validated-profiles/{profile-id})
	GetValidatedProfile(ctx context.Context, request GetValidatedProfileRequestObject) (UpstreamProfileGetProfileRequestObject, error)
}

type StrictUpstreamInterface interface {
	Profile() http.HandlerFunc
}

type StrictOperationsMap[T any] struct {
	GetProfile          T
	PutProfile          T
	GetValidatedProfile T
}

func (s StrictOperationsMap[T]) Get(opid string) (t T, found bool) {
	switch opid {
	case "GetProfile":
		return s.GetProfile, true

	case "PutProfile":
		return s.PutProfile, true

	case "GetValidatedProfile":
		return s.GetValidatedProfile, true

	}

	return t, false
}

func (s StrictOperationsMap[T]) ToMap() (m map[string]T) {
	return map[string]T{
		"GetProfile":          s.GetProfile,
		"PutProfile":          s.PutProfile,
		"GetValidatedProfile": s.GetValidatedProfile,
	}
}

type StrictHandlerFunc = strictecho.StrictEchoHandlerFunc
type StrictMiddlewareFunc = strictecho.StrictEchoMiddlewareFunc

func NewStrictHandler(ssi StrictServerInterface, sui StrictUpstreamInterface, middlewares []StrictMiddlewareFunc) ServerInterface {
	return &strictHandler{ssi: ssi, sui: sui, middlewares: middlewares}
}

type strictHandler struct {
	ssi         StrictServerInterface
	sui         StrictUpstreamInterface
	middlewares []StrictMiddlewareFunc
}

// GetProfile operation middleware
func (sh *strictHandler) GetProfile(ctx echo.Context, profileId ProfileProfileID, params GetProfileParams) error {
	var request GetProfileRequestObject

	request.ProfileId = profileId
	request.Params = params
	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.GetProfile(ctx.Request().Context(), request.(GetProfileRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "GetProfile")
	}

	obj, err := handler(ctx, request)
	if err != nil {
		return err
	}

	outreq, err := obj.(strictRequest).ToRequest(ctx.Request())
	if err != nil {
		return err
	}

	sh.sui.Profile()(ctx.Response(), outreq)

	return nil
}

// PutProfile operation middleware
func (sh *strictHandler) PutProfile(ctx echo.Context, profileId ProfileProfileID, params PutProfileParams) error {
	var request PutProfileRequestObject

	request.ProfileId = profileId
	request.Params = params
	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.PutProfile(ctx.Request().Context(), request.(PutProfileRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "PutProfile")
	}

	obj, err := handler(ctx, request)
	if err != nil {
		return err
	}

	outreq, err := obj.(strictRequest).ToRequest(ctx.Request())
	if err != nil {
		return err
	}

	sh.sui.Profile()(ctx.Response(), outreq)

	return nil
}

// GetValidatedProfile operation middleware
func (sh *strictHandler) GetValidatedProfile(ctx echo.Context, profileId ProfileProfileID, params GetValidatedProfileParams) error {
	var request GetValidatedProfileRequestObject

	request.ProfileId = profileId
	request.Params = params
	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.GetValidatedProfile(ctx.Request().Context(), request.(GetValidatedProfileRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "GetValidatedProfile")
	}

	obj, err := handler(ctx, request)
	if err != nil {
		return err
	}

	outreq, err := obj.(strictRequest).ToRequest(ctx.Request())
	if err != nil {
		return err
	}

	sh.sui.Profile()(ctx.Response(), outreq)

	return nil
}

/* ignored */

// UpstreamProfileProfile defines model for UpstreamProfileProfile.
type UpstreamProfileProfile struct {
	Dob      UpstreamProfileZeroableTime   `json:"dob,omitempty"`
	Email    UpstreamProfileZeroableString `json:"email,omitempty"`
	Id       UpstreamProfileUUID           `json:"id,omitempty"`
	Name     UpstreamProfileZeroableString `json:"name,omitempty"`
	Nin      UpstreamProfileZeroableString `json:"nin,omitempty"`
	Phone    UpstreamProfileZeroableString `json:"phone,omitempty"`
	TenantId UpstreamProfileUUID           `json:"tenant_id,omitempty"`
}

// UpstreamProfileUUID defines model for UpstreamProfileUUID.
type UpstreamProfileUUID = openapi_types.UUID

// UpstreamProfileZeroableString defines model for UpstreamProfileZeroableString.
type UpstreamProfileZeroableString = string

// UpstreamProfileZeroableTime defines model for UpstreamProfileZeroableTime.
type UpstreamProfileZeroableTime = time.Time

// UpstreamProfileProfileID defines model for UpstreamProfileProfileID.
type UpstreamProfileProfileID = UpstreamProfileUUID

// UpstreamProfileError defines model for UpstreamProfileError.
type UpstreamProfileError struct {
	Id UpstreamProfileUUID `json:"id,omitempty"`
}

// UpstreamProfileGetProfileParams defines parameters for UpstreamProfileGetProfile.
type UpstreamProfileGetProfileParams struct {
	SomeQuery *string `form:"some-query,omitempty" json:"some-query,omitempty"`
}

// UpstreamProfilePutProfileParams defines parameters for UpstreamProfilePutProfile.
type UpstreamProfilePutProfileParams struct {
	SomeQuery *string `form:"some-query,omitempty" json:"some-query,omitempty"`
}

/* ignored */

/* ignored */

/* ignored */

/* ignored */

/* ignored */

type UpstreamProfileGetProfileRequestObject struct {
	TenantId  UpstreamProfileUUID      `json:"tenant-id"`
	ProfileId UpstreamProfileProfileID `json:"profile-id"`
	Params    UpstreamProfileGetProfileParams
}

func (r UpstreamProfileGetProfileRequestObject) ToRequest(base *http.Request) (*http.Request, error) {

	tenantId := r.TenantId
	profileId := r.ProfileId
	params := r.Params

	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "tenant-id", runtime.ParamLocationPath, tenantId)
	if err != nil {
		return nil, err
	}

	var pathParam1 string

	pathParam1, err = runtime.StyleParamWithLocation("simple", false, "profile-id", runtime.ParamLocationPath, profileId)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/tenants/%s/profiles/%s", pathParam0, pathParam1)
	queryURL, err := url.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	queryValues := queryURL.Query()

	if params.SomeQuery != nil {

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "some-query", runtime.ParamLocationQuery, *params.SomeQuery); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

	}

	queryURL.RawQuery = queryValues.Encode()

	req := base.Clone(base.Context())
	req.URL = queryURL

	return req, nil
}

type UpstreamProfilePutProfileRequestObject struct {
	TenantId  UpstreamProfileUUID      `json:"tenant-id"`
	ProfileId UpstreamProfileProfileID `json:"profile-id"`
	Params    UpstreamProfilePutProfileParams
}

func (r UpstreamProfilePutProfileRequestObject) ToRequest(base *http.Request) (*http.Request, error) {

	tenantId := r.TenantId
	profileId := r.ProfileId
	params := r.Params

	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "tenant-id", runtime.ParamLocationPath, tenantId)
	if err != nil {
		return nil, err
	}

	var pathParam1 string

	pathParam1, err = runtime.StyleParamWithLocation("simple", false, "profile-id", runtime.ParamLocationPath, profileId)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/tenants/%s/profiles/%s", pathParam0, pathParam1)
	queryURL, err := url.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	queryValues := queryURL.Query()

	if params.SomeQuery != nil {

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "some-query", runtime.ParamLocationQuery, *params.SomeQuery); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

	}

	queryURL.RawQuery = queryValues.Encode()

	req := base.Clone(base.Context())
	req.URL = queryURL

	return req, nil
}

/* ignored */
