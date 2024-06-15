package testgen_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testgen"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

var _ testgen.StrictUpstreamInterface = ProxyImpl{}
var _ testgen.StrictServerInterface = ServerImpl{}

type ProxyImpl struct {
	profile *httputil.ReverseProxy
}

func (p ProxyImpl) Profile() http.HandlerFunc {
	return p.profile.ServeHTTP
}

type ServerImpl struct {
	tenantID uuid.UUID
}

// GetProfile implements testgen.StrictServerInterface.
func (s ServerImpl) GetProfile(ctx context.Context, request testgen.GetProfileRequestObject) (testgen.UpstreamProfileGetProfileRequestObject, error) {
	return testgen.UpstreamProfileGetProfileRequestObject{
		TenantId:  s.tenantID,
		ProfileId: request.ProfileId,
		Params: testgen.UpstreamProfileGetProfileParams{
			SomeQuery: request.Params.SomeQuery,
		},
	}, nil
}

// GetValidatedProfile implements testgen.StrictServerInterface.
func (s ServerImpl) GetValidatedProfile(ctx context.Context, request testgen.GetValidatedProfileRequestObject) (testgen.UpstreamProfileGetProfileRequestObject, error) {
	return testgen.UpstreamProfileGetProfileRequestObject{
		TenantId:  s.tenantID,
		ProfileId: request.ProfileId,
		Params: testgen.UpstreamProfileGetProfileParams{
			SomeQuery: request.Params.SomeQuery,
		},
	}, nil
}

// PutProfile implements testgen.StrictServerInterface.
func (s ServerImpl) PutProfile(ctx context.Context, request testgen.PutProfileRequestObject) (testgen.UpstreamProfilePutProfileRequestObject, error) {
	return testgen.UpstreamProfilePutProfileRequestObject{
		TenantId:  s.tenantID,
		ProfileId: request.ProfileId,
		Params: testgen.UpstreamProfilePutProfileParams{
			SomeQuery: request.Params.SomeQuery,
		},
	}, nil
}

func TestProxy(t *testing.T) {
	var receivedURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
	}))

	u, _ := url.Parse(ts.URL)
	p := ProxyImpl{
		profile: httputil.NewSingleHostReverseProxy(u),
	}
	s := ServerImpl{tenantID: uuid.New()}

	e := echo.New()
	sh := testgen.NewStrictHandler(s, p, nil)
	testgen.RegisterHandlers(e, sh)

	id := uuid.NewString()
	testtable := []struct {
		i string
		o string
	}{
		{
			i: "/profiles/" + id,
			o: "/tenants/" + s.tenantID.String() + "/profiles/" + id,
		},
	}

	for _, d := range testtable {
		req := httptest.NewRequest(http.MethodGet, d.i, nil)
		res := httptest.NewRecorder()
		e.ServeHTTP(res, req)

		assert.Equal(t, d.o, receivedURL)

	}

}