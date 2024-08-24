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
				func(context.Context) (bool, error) { return true, nil },
				e.AND(
					func(context.Context) (bool, error) { return true, nil },
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
			ctx.SetRequest(
				ctx.Request().WithContext(
					ctxWithAuthzExpectation(ctx.Request().Context(), func(ae *authzExpectations) {
						ae.tenantID, _ = ctx.Request().Context().Value(ctxTenantID{}).(uuid.UUID)
					})))

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

type authzExpectation func(ctx context.Context) (bool, error)

type authzExpectations struct {
	tenantID uuid.UUID

	invoked bool
	result  bool
	err     error
}

func (a *authzExpectations) OR(reqs ...authzExpectation) authzExpectation {
	if a == nil {
		a.False()
	}

	return func(ctx context.Context) (oks bool, errs error) {
		for _, step := range reqs {
			ok, err := step(ctx)
			if !ok || err != nil {
				errs = errors.Join(errs, err)
				continue
			}

			oks = true
		}
		return
	}
}

func (a *authzExpectations) AND(reqs ...authzExpectation) authzExpectation {
	if a == nil {
		a.False()
	}

	return func(ctx context.Context) (bool, error) {
		for _, step := range reqs {
			ok, err := step(ctx)
			if !ok || err != nil {
				return ok, err
			}
		}
		return true, nil
	}
}

func (a *authzExpectations) False() authzExpectation {
	return func(context.Context) (bool, error) { return false, nil }
}

func (a *authzExpectations) tenantIDNotZero() authzExpectation {
	if a == nil {
		a.False()
	}

	return func(context.Context) (bool, error) {
		return a.tenantID != uuid.UUID{}, nil
	}
}

func (a *authzExpectations) ProfileIDNotZero(profileID uuid.UUID) authzExpectation {
	if a == nil {
		a.False()
	}

	return func(context.Context) (bool, error) {
		return profileID != uuid.UUID{}, nil
	}
}

func ctxWithAuthzExpectation(ctx context.Context, init func(*authzExpectations)) context.Context {
	a := &authzExpectations{}
	init(a)

	return context.WithValue(ctx, authzExpectations{}, a)

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
		v.result, v.err = exp(ctx)
		if !v.result || v.err != nil {
			return v.result, v.err
		}
	}
	v.result = true

	return v.result, nil
}
