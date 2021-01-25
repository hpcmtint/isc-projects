package restservice

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbsession "isc.org/stork/server/database/session"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test"
)

// Check if fileServerMiddleware works and handles requests correctly.
func TestFileServerMiddleware(t *testing.T) {
	apiRequestReceived := false
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiRequestReceived = true
	})

	handler := fileServerMiddleware(apiHandler, "./non-existing-static/")

	// let request some static file, as it does not exist 404 code should be returned
	req := httptest.NewRequest("GET", "http://localhost/abc", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.EqualValues(t, 404, resp.StatusCode)
	require.False(t, apiRequestReceived)

	// let request some API URL, it should be forwarded to apiHandler
	req = httptest.NewRequest("GET", "http://localhost/api/users", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.True(t, apiRequestReceived)

	// request for swagger.json also should be forwarded to apiHandler
	req = httptest.NewRequest("GET", "http://localhost/swagger.json", nil)
	w = httptest.NewRecorder()
	apiRequestReceived = false
	handler.ServeHTTP(w, req)
	require.True(t, apiRequestReceived)
}

// Check if InnerMiddleware works.
func TestInnerMiddleware(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	settings := RestAPISettings{}
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}
	rapi, err := NewRestAPI(&settings, dbSettings, db, fa, fec, nil)
	require.NoError(t, err)
	sm, err := dbsession.NewSessionMgr(&rapi.DBSettings.BaseDatabaseSettings)
	require.NoError(t, err)
	rapi.SessionManager = sm

	handler := rapi.InnerMiddleware(nil)
	require.NotNil(t, handler)
}

// Check if fileServerMiddleware works and handles requests correctly.
func TestSSEMiddleware(t *testing.T) {
	requestReceived := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
	})

	fec := &storktest.FakeEventCenter{}

	handler := sseMiddleware(nextHandler, fec)

	// let request sse
	req := httptest.NewRequest("GET", "http://localhost/sse", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.EqualValues(t, 200, resp.StatusCode)
	require.False(t, requestReceived)

	// let request something else than sse, it should be forwarded to nextHandler
	req = httptest.NewRequest("GET", "http://localhost/api/users", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.True(t, requestReceived)
}

// Check if agentInstallerMiddleware works and handles requests correctly.
func TestAgentInstallerMiddleware(t *testing.T) {
	requestReceived := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
	})

	tmpDir, err := ioutil.TempDir("", "mdlw")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	handler := agentInstallerMiddleware(nextHandler, tmpDir)

	// let do some request but when there is no folder with static content
	req := httptest.NewRequest("GET", "http://localhost/stork-install-agent.sh", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	resp.Body.Close()
	require.EqualValues(t, 500, resp.StatusCode)
	require.False(t, requestReceived)

	// prepare folders
	os.Mkdir(path.Join(tmpDir, "assets"), 0755)
	os.Mkdir(path.Join(tmpDir, "assets/pkgs"), 0755)

	// let do some request
	req = httptest.NewRequest("GET", "http://localhost/stork-install-agent.sh", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp = w.Result()
	resp.Body.Close()
	require.EqualValues(t, 500, resp.StatusCode)
	require.False(t, requestReceived)

	// let request something else, it should be forwarded to nextHandler
	req = httptest.NewRequest("GET", "http://localhost/api/users", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.True(t, requestReceived)
}
