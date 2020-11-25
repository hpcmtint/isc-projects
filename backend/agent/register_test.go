package agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/pki"
)

// Check if registration works.
func TestRegister(t *testing.T) {
	// prepare temp dir for cert files
	tmpDir, err := ioutil.TempDir("", "reg")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	os.Mkdir(path.Join(tmpDir, "certs"), 0755)
	os.Mkdir(path.Join(tmpDir, "tokens"), 0755)

	// redefined consts with paths to cert files
	KeyPEMFile = path.Join(tmpDir, "certs/key.pem")
	CertPEMFile = path.Join(tmpDir, "certs/cert.pem")
	RootCAFile = path.Join(tmpDir, "certs/ca.pem")
	AgentTokenFile = path.Join(tmpDir, "tokens/agent-token.txt")

	// register arguments
	serverToken := "serverToken"
	agentAddr := "1.2.3.4"
	agentPort := 8080
	regenCerts := false
	retry := false

	// internal http server for testing
	require.NoError(t, err)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("URL: %v\n", r.URL.Path)

		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		fmt.Printf("BODY: %v\n", string(body))
		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		if r.URL.Path == "/api/machines" {
			require.EqualValues(t, req["address"].(string), agentAddr)
			require.EqualValues(t, int(req["agentPort"].(float64)), agentPort)
			serverTokenRcvd := req["serverToken"].(string)
			agentToken := req["agentToken"].(string)
			if serverToken == "" {
				require.NotEmpty(t, agentToken)
			} else {
				require.EqualValues(t, serverToken, serverTokenRcvd)
				require.Empty(t, agentToken)
			}

			agentCSR := []byte(req["agentCSR"].(string))
			require.NotEmpty(t, agentCSR)

			_, rootKeyPEM, _, rootCertPEM, err := pki.GenCACert(1)
			require.NoError(t, err)
			agentCertPEM, _, paramsErr, innerErr := pki.SignCert(agentCSR, 2, rootCertPEM, rootKeyPEM)
			require.NoError(t, paramsErr)
			require.NoError(t, innerErr)

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id":           10,
				"serverCACert": string(rootCertPEM),
				"agentCert":    string(agentCertPEM),
			}
			json.NewEncoder(w).Encode(resp)
		}

		if strings.HasSuffix(r.URL.Path, "/ping") {
			serverTokenRcvd := req["serverToken"].(string)
			agentToken := req["agentToken"].(string)
			if serverToken == "" {
				require.NotEmpty(t, agentToken)
			} else {
				require.EqualValues(t, serverToken, serverTokenRcvd)
				require.Empty(t, agentToken)
			}

			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"id": 10,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer ts.Close()

	serverURL := ts.URL

	// register with server token
	res := Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenCerts, retry)
	require.True(t, res)

	// register with agent token
	serverToken = ""
	res = Register(serverURL, serverToken, agentAddr, fmt.Sprintf("%d", agentPort), regenCerts, retry)
	require.True(t, res)
}
