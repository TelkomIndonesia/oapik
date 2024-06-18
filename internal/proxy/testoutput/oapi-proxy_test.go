package testoutput_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"testoutput"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	strictecho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
	"github.com/stretchr/testify/assert"
)

var _ testoutput.StrictUpstreamInterface = ProxyImpl{}
var _ testoutput.StrictServerInterface = ServerImpl{}

type ProxyImpl struct {
	profile *httputil.ReverseProxy
}

func (p ProxyImpl) Profile() http.HandlerFunc {
	return p.profile.ServeHTTP
}

type ServerImpl struct {
}

// ProfileGetProfile implements testoutput.StrictServerInterface.
func (s ServerImpl) ProfileGetProfile(ctx context.Context, request testoutput.ProfileGetProfileRequestObject) (testoutput.ProfileGetProfileRequestObject, error) {
	panic("unimplemented")
}

// GetProfile implements testoutput.StrictServerInterface.
func (s ServerImpl) GetProfile(ctx context.Context, request testoutput.GetProfileRequestObject) (testoutput.UpstreamProfileGetProfileRequestObject, error) {
	return testoutput.UpstreamProfileGetProfileRequestObject{
		TenantId:  ctx.Value(ctxTenantID{}).(uuid.UUID),
		ProfileId: request.ProfileId,
		Params: testoutput.UpstreamProfileGetProfileParams{
			SomeQuery: request.Params.SomeQuery,
		},
	}, nil
}

// GetValidatedProfile implements testoutput.StrictServerInterface.
func (s ServerImpl) GetValidatedProfile(ctx context.Context, request testoutput.GetValidatedProfileRequestObject) (testoutput.UpstreamProfileGetProfileRequestObject, error) {
	return testoutput.UpstreamProfileGetProfileRequestObject{
		TenantId:  ctx.Value(ctxTenantID{}).(uuid.UUID),
		ProfileId: request.ProfileId,
		Params: testoutput.UpstreamProfileGetProfileParams{
			SomeQuery: request.Params.SomeQuery,
		},
	}, nil
}

// PutProfile implements testoutput.StrictServerInterface.
func (s ServerImpl) PutProfile(ctx context.Context, request testoutput.PutProfileRequestObject) (testoutput.UpstreamProfilePutProfileRequestObject, error) {
	return testoutput.UpstreamProfilePutProfileRequestObject{
		TenantId:  ctx.Value(ctxTenantID{}).(uuid.UUID),
		ProfileId: request.ProfileId,
		Params: testoutput.UpstreamProfilePutProfileParams{
			SomeQuery: request.Params.SomeQuery,
		},
	}, nil
}

type ctxTenantID struct{}

func TestProxy(t *testing.T) {
	var receivedURL string
	profileServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
	}))

	u, _ := url.Parse(profileServer.URL)
	proxyImpl := ProxyImpl{
		profile: httputil.NewSingleHostReverseProxy(u),
	}
	serverImpl := ServerImpl{}

	tenantID := uuid.New()
	insertTenantID := func(f strictecho.StrictEchoHandlerFunc, operationID string) strictecho.StrictEchoHandlerFunc {
		return func(ctx echo.Context, request interface{}) (response interface{}, err error) {
			ctx.SetRequest(
				ctx.Request().WithContext(
					context.WithValue(ctx.Request().Context(),
						ctxTenantID{}, tenantID,
					)))

			return f(ctx, request)
		}
	}

	t.Run("Standard", func(t *testing.T) {
		e := echo.New()
		sh := testoutput.NewStrictHandler(serverImpl, proxyImpl, []strictecho.StrictEchoMiddlewareFunc{insertTenantID})
		testoutput.RegisterHandlers(e, sh)

		id := uuid.NewString()
		testtable := []struct {
			name string
			i    string
			o    string
		}{
			{
				name: "GetProfile",
				i:    "/profiles/" + id,
				o:    "/tenants/" + tenantID.String() + "/profiles/" + id,
			},
			{
				name: "GetValidatedProfile",
				i:    "/validated-profiles/" + id,
				o:    "/tenants/" + tenantID.String() + "/profiles/" + id,
			},
		}

		for _, d := range testtable {
			t.Run(d.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, d.i, nil)
				res := httptest.NewRecorder()
				e.ServeHTTP(res, req)

				assert.Equal(t, d.o, receivedURL)
			})
		}
	})

	t.Run("SelectivePassthroughMiddleware", func(t *testing.T) {
		selectivePasstrough := func() testoutput.StrictMiddlewareFunc {
			excludes := testoutput.StrictOperationsMap[bool]{
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

		e := echo.New()
		sh := testoutput.NewStrictHandler(serverImpl, proxyImpl, []strictecho.StrictEchoMiddlewareFunc{insertTenantID, selectivePasstrough()})
		testoutput.RegisterHandlers(e, sh)

		id := uuid.NewString()
		testtable := []struct {
			name string
			i    string
			o    string
		}{
			{
				name: "Passthrough",
				i:    "/tenants/" + tenantID.String() + "/profiles/" + id,
				o:    "/tenants/" + tenantID.String() + "/profiles/" + id,
			},
			{
				name: "NotPassthrough",
				i:    "/validated-profiles/" + id,
				o:    "/tenants/" + tenantID.String() + "/profiles/" + id,
			},
		}

		for _, d := range testtable {
			t.Run(d.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, d.i, nil)
				res := httptest.NewRecorder()
				e.ServeHTTP(res, req)

				assert.Equal(t, d.o, receivedURL)
			})
		}
	})

}
