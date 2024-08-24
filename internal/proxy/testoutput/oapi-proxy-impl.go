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
	authzAssertionExpect(ctx, func(a *authzAssertions) []authzAssertionFunc {
		return []authzAssertionFunc{
			a.ProfileIDNotZero(request.ProfileId),
			a.OR(
				func() (bool, error) { return true, nil },
				a.AND(
					func() (bool, error) { return true, nil },
					func() (bool, error) { return false, nil },
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

func insertTenantIDMiddleware(tenantID uuid.UUID) strictecho.StrictEchoMiddlewareFunc {
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
			a := authzAssertions{
				tenantID: ctx.Request().Context().Value(ctxTenantID{}).(uuid.UUID),
			}
			ctx.SetRequest(ctx.Request().WithContext(a.Attach(ctx.Request().Context())))

			// exec handler
			res, err := f(ctx, request)
			if err != nil {
				return nil, err
			}

			// assert permission result
			if a.err != nil {
				return nil, err
			}
			if !a.result {
				return nil, echo.NewHTTPError(http.StatusForbidden, "forbidden")
			}
			return res, err
		}
	}
}

type authzAssertionFunc func() (bool, error)

type authzAssertions struct {
	tenantID uuid.UUID

	result bool
	err    error
}

func (a *authzAssertions) Attach(ctx context.Context) context.Context {
	if a == nil {
		return ctx
	}

	return context.WithValue(ctx, authzAssertions{}, a)
}

func (a *authzAssertions) Expect(f ...authzAssertionFunc) (bool, error) {
	if a == nil {
		return false, nil
	}

	if a.tenantID == (uuid.UUID{}) {
		return false, nil
	}

	for _, req := range f {
		a.result, a.err = req()
		if !a.result || a.err != nil {
			return a.result, a.err
		}
	}

	a.result = true
	return a.result, nil
}

func (a *authzAssertions) OR(reqs ...func() (bool, error)) authzAssertionFunc {
	if a == nil {
		return func() (bool, error) { return false, nil }
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

func (a *authzAssertions) AND(reqs ...func() (bool, error)) authzAssertionFunc {
	if a == nil {
		return func() (bool, error) { return false, nil }
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

func (a *authzAssertions) ProfileIDNotZero(profileID uuid.UUID) authzAssertionFunc {
	if a == nil {
		return func() (bool, error) { return false, nil }
	}

	return func() (bool, error) {
		return profileID != uuid.UUID{}, nil
	}
}

func authzAssertionExpect(ctx context.Context, f func(*authzAssertions) []authzAssertionFunc) (bool, error) {
	v, _ := (ctx.Value(authzAssertions{})).(*authzAssertions)
	return v.Expect(f(v)...)
}
