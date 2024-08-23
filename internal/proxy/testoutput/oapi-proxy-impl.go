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
	a := authzAssertionFromContext(ctx)
	a.expect(ctx.Value(ctxTenantID{}).(uuid.UUID),
		a.profileIDNotZero(request.ProfileId),
		a.or(
			a.profileIDNotZero(request.ProfileId),
			func() (bool, error) { return false, nil },
		),
	)

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
			a := authzAssertion{}
			ctx.SetRequest(ctx.Request().WithContext(a.attach(ctx.Request().Context())))

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

type authzAssertion struct {
	result bool
	err    error
}

func (a *authzAssertion) attach(ctx context.Context) context.Context {
	if a == nil {
		return ctx
	}

	return context.WithValue(ctx, authzAssertion{}, a)
}

func (a *authzAssertion) expect(tenantID uuid.UUID, reqs ...func() (bool, error)) (bool, error) {
	if a == nil {
		return false, nil
	}

	if tenantID == (uuid.UUID{}) {
		return false, nil
	}

	for _, req := range reqs {
		a.result, a.err = req()
		if !a.result || a.err != nil {
			return a.result, a.err
		}
	}

	a.result = true
	return a.result, nil
}

func (a *authzAssertion) profileIDNotZero(profileID uuid.UUID) func() (bool, error) {
	if a == nil {
		return func() (bool, error) { return false, nil }
	}

	return func() (bool, error) {
		return profileID != uuid.UUID{}, nil
	}
}

func (a *authzAssertion) or(reqs ...func() (bool, error)) func() (bool, error) {
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

func authzAssertionFromContext(ctx context.Context) *authzAssertion {
	v, _ := (ctx.Value(authzAssertion{})).(*authzAssertion)
	return v
}
