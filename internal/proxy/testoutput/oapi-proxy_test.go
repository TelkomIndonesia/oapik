package testoutput

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	strictecho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	profileServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		io.WriteString(w, r.URL.String())
	}))

	u, _ := url.Parse(profileServer.URL)
	proxyImpl := proxyImpl{
		profile: httputil.NewSingleHostReverseProxy(u),
	}
	serverImpl := serverImpl{}

	tenantID := uuid.New()

	t.Run("Standard", func(t *testing.T) {
		e := echo.New()
		sh := NewStrictHandler(serverImpl, proxyImpl, []strictecho.StrictEchoMiddlewareFunc{injectTenantIDMiddleware(tenantID)})
		RegisterHandlers(e, sh)

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
				assert.Equal(t, http.StatusAccepted, res.Code)
				rurl, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.Equal(t, d.o, string(rurl))
			})
		}
	})

	t.Run("SelectivePassthroughMiddleware", func(t *testing.T) {
		e := echo.New()
		sh := NewStrictHandler(serverImpl, proxyImpl, []strictecho.StrictEchoMiddlewareFunc{
			selectivePasstroughMiddleware(),
			injectTenantIDMiddleware(tenantID),
		})
		RegisterHandlers(e, sh)

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

				assert.Equal(t, http.StatusAccepted, res.Code)
				rurl, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.Equal(t, d.o, string(rurl))
			})
		}
	})

	t.Run("Authz", func(t *testing.T) {
		e := echo.New()
		sh := NewStrictHandler(serverImpl, proxyImpl, []strictecho.StrictEchoMiddlewareFunc{
			authz(),
			injectTenantIDMiddleware(tenantID),
		})
		RegisterHandlers(e, sh)

		id := uuid.NewString()
		testtable := []struct {
			name   string
			path   string
			method string
			code   int
		}{
			{
				name:   "Authorized",
				path:   "/profiles/" + id,
				method: http.MethodGet,
				code:   http.StatusAccepted,
			},
			{
				name:   "AuthorizedWithoutAdditionalExpectation",
				path:   "/profiles/" + id,
				method: http.MethodPut,
				code:   http.StatusAccepted,
			},
			{
				name:   "Unauthorized",
				path:   "/validated-profiles/" + id,
				method: http.MethodGet,
				code:   http.StatusForbidden,
			},
		}

		for _, d := range testtable {
			t.Run(d.name, func(t *testing.T) {
				req := httptest.NewRequest(d.method, d.path, nil)
				res := httptest.NewRecorder()
				e.ServeHTTP(res, req)

				assert.Equal(t, d.code, res.Code)
			})
		}
	})

	t.Run("OperationData", func(t *testing.T) {
		assert.Equal(t, "profile", StrictOperationsData.GetProfile.Extension.Data.ProxyName)
	})
}
