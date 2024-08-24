package testoutput

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	strictecho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
)

var _ StrictUpstreamInterface = proxyImpl{}
var _ StrictServerInterface = serverImpl{}

type proxyImpl struct {
	profile *httputil.ReverseProxy
}

func (p proxyImpl) Profile() http.HandlerFunc {
	return p.profile.ServeHTTP
}

type serverImpl struct{}

// ProfileGetProfile implements StrictServerInterface.
func (s serverImpl) ProfileGetProfile(ctx context.Context, request ProfileGetProfileRequestObject) (ProfileGetProfileRequestObject, error) {
	panic("unimplemented")
}

// GetProfile implements StrictServerInterface.
func (s serverImpl) GetProfile(ctx context.Context, request GetProfileRequestObject) (UpstreamProfileGetProfileRequestObject, error) {
	authzExpect(ctx, func(e *authzExpectations) []authzExpectation {
		return []authzExpectation{
			e.ProfileIDNotZero(request.ProfileId),
			e.OR(
				func() (bool, error) { return true, nil },
				e.AND(
					func() (bool, error) { return true, nil },
					e.False(),
				),
			),
		}
	})

	return UpstreamProfileGetProfileRequestObject{
		TenantId:  ctx.Value(ctxTenantID{}).(uuid.UUID),
		ProfileId: request.ProfileId,
		Params: UpstreamProfileGetProfileParams{
			SomeQuery: request.Params.SomeQuery,
		},
	}, nil
}

// GetValidatedProfile implements StrictServerInterface.
func (s serverImpl) GetValidatedProfile(ctx context.Context, request GetValidatedProfileRequestObject) (UpstreamProfileGetProfileRequestObject, error) {
	authzExpect(ctx, func(ae *authzExpectations) []authzExpectation { return []authzExpectation{ae.False()} })
	return UpstreamProfileGetProfileRequestObject{
		TenantId:  ctx.Value(ctxTenantID{}).(uuid.UUID),
		ProfileId: request.ProfileId,
		Params: UpstreamProfileGetProfileParams{
			SomeQuery: request.Params.SomeQuery,
		},
	}, nil
}

// PutProfile implements StrictServerInterface.
func (s serverImpl) PutProfile(ctx context.Context, request PutProfileRequestObject) (UpstreamProfilePutProfileRequestObject, error) {
	return UpstreamProfilePutProfileRequestObject{
		TenantId:  ctx.Value(ctxTenantID{}).(uuid.UUID),
		ProfileId: request.ProfileId,
		Params: UpstreamProfilePutProfileParams{
			SomeQuery: request.Params.SomeQuery,
		},
	}, nil
}

type ctxTenantID struct{}

func injectTenantIDMiddleware(tenantID uuid.UUID) strictecho.StrictEchoMiddlewareFunc {
	return func(f strictecho.StrictEchoHandlerFunc, operationID string) strictecho.StrictEchoHandlerFunc {
		return func(ctx echo.Context, request interface{}) (response interface{}, err error) {
			ctx.SetRequest(
				ctx.Request().WithContext(
					context.WithValue(ctx.Request().Context(),
						ctxTenantID{}, tenantID,
					)))

			return f(ctx, request)
		}
	}
}

func selectivePasstroughMiddleware() strictecho.StrictEchoMiddlewareFunc {
	excludes := StrictOperationsMap[bool]{
		GetProfile:          true,
		PutProfile:          true,
		GetValidatedProfile: true,
	}
	return func(f strictecho.StrictEchoHandlerFunc, operationID string) strictecho.StrictEchoHandlerFunc {
		if yes, _ := excludes.Get(operationID); yes {
			return f
		}

		return func(ctx echo.Context, request interface{}) (response interface{}, err error) {
			return request, err
		}
	}
}

func authz() strictecho.StrictEchoMiddlewareFunc {
	return func(f strictecho.StrictEchoHandlerFunc, operationID string) strictecho.StrictEchoHandlerFunc {
		return func(ctx echo.Context, request interface{}) (response interface{}, err error) {
			// init expectations
			exp := authzExpectations{}
			exp.tenantID, _ = ctx.Request().Context().Value(ctxTenantID{}).(uuid.UUID)
			ctx.SetRequest(ctx.Request().WithContext(exp.Attach(ctx.Request().Context())))

			// exec handler
			res, err := f(ctx, request)
			if err != nil {
				return nil, err
			}

			// verify expectations result
			result, err := authzExpect(ctx.Request().Context(), nil)
			if err != nil {
				return nil, err
			}
			if !result {
				return nil, echo.NewHTTPError(http.StatusForbidden, "forbidden")
			}
			return res, err
		}
	}
}

type authzExpectation func() (bool, error)

type authzExpectations struct {
	tenantID uuid.UUID

	invoked bool
	result  bool
	err     error
}

func (a *authzExpectations) Attach(ctx context.Context) context.Context {
	if a == nil {
		return ctx
	}

	return context.WithValue(ctx, authzExpectations{}, a)
}

func (a *authzExpectations) OR(reqs ...func() (bool, error)) authzExpectation {
	if a == nil {
		a.False()
	}

	return func() (oks bool, errs error) {
		for _, step := range reqs {
			ok, err := step()
			if !ok || err != nil {
				errs = errors.Join(errs, err)
				continue
			}

			oks = true
		}
		return
	}
}

func (a *authzExpectations) AND(reqs ...func() (bool, error)) authzExpectation {
	if a == nil {
		a.False()
	}

	return func() (bool, error) {
		for _, step := range reqs {
			ok, err := step()
			if !ok || err != nil {
				return ok, err
			}
		}
		return true, nil
	}
}

func (a *authzExpectations) False() authzExpectation {
	return func() (bool, error) { return false, nil }
}

func (a *authzExpectations) tenantIDNotZero() authzExpectation {
	if a == nil {
		a.False()
	}

	return func() (bool, error) {
		return a.tenantID != uuid.UUID{}, nil
	}
}

func (a *authzExpectations) ProfileIDNotZero(profileID uuid.UUID) authzExpectation {
	if a == nil {
		a.False()
	}

	return func() (bool, error) {
		return profileID != uuid.UUID{}, nil
	}
}

func authzExpect(ctx context.Context, addExp func(*authzExpectations) []authzExpectation) (bool, error) {
	v, _ := (ctx.Value(authzExpectations{})).(*authzExpectations)
	if v == nil {
		return false, nil
	}

	if v.invoked {
		return v.result, v.err
	}

	// required expectations
	exps := []authzExpectation{v.tenantIDNotZero()}
	// additional expectations
	if addExp != nil {
		exps = append(exps, addExp(v)...)
	}

	v.invoked = true
	for _, exp := range exps {
		v.result, v.err = exp()
		if !v.result || v.err != nil {
			return v.result, v.err
		}
	}
	v.result = true

	return v.result, nil
}
