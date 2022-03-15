package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Showmax/go-fqdn"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"

	"isc.org/stork/pki"
	storkutil "isc.org/stork/util"
)

// Paths pointing to agent's key and cert, and CA cert from server,
// and agent token generated by agent.
// They are being modified by tests so need to be writable.
var (
	KeyPEMFile     = "/var/lib/stork-agent/certs/key.pem"          // nolint:gochecknoglobals
	CertPEMFile    = "/var/lib/stork-agent/certs/cert.pem"         // nolint:gochecknoglobals
	RootCAFile     = "/var/lib/stork-agent/certs/ca.pem"           // nolint:gochecknoglobals
	AgentTokenFile = "/var/lib/stork-agent/tokens/agent-token.txt" // nolint:gochecknoglobals,gosec
)

// Prompt user for server token. If user hits enter key then empty
// string is returned.
func promptUserForServerToken() (string, error) {
	fmt.Printf(">>>> Server access token (optional): ")
	serverToken, err := term.ReadPassword(0)
	fmt.Print("\n")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(serverToken)), nil
}

// Get agent's address and port from user if not provided via command line options.
func getAgentAddrAndPortFromUser(agentAddr, agentPort string) (string, int, error) {
	if agentAddr == "" {
		agentAddrTip, err := fqdn.FqdnHostname()
		msg := ">>>> IP address or FQDN of the host with Stork Agent (the Stork Server will use it to connect to the Stork Agent)"
		if err != nil {
			agentAddrTip = ""
			msg += ": "
		} else {
			msg += fmt.Sprintf(" [%s]: ", agentAddrTip)
		}
		fmt.Print(msg)
		fmt.Scanln(&agentAddr)
		agentAddr = strings.TrimSpace(agentAddr)
		if agentAddr == "" {
			agentAddr = agentAddrTip
		}
	}

	if agentPort == "" {
		fmt.Printf(">>>> Port number that Stork Agent will use to listen on [8080]: ")
		fmt.Scanln(&agentPort)
		agentPort = strings.TrimSpace(agentPort)
		if agentPort == "" {
			agentPort = "8080"
		}
	}

	agentPortInt, err := strconv.Atoi(agentPort)
	if err != nil {
		log.Errorf("%s is not a valid agent port number: %s", agentPort, err)
		return "", 0, err
	}
	return agentAddr, agentPortInt, nil
}

// Write agent file. Used to save key or certs.
// They are sensitive so permissions are set to 0600.
func writeAgentFile(path string, content []byte) error {
	_, err := os.Stat(path)
	if os.IsExist(err) {
		err = os.Remove(path)
		if err != nil {
			return err
		}
	}

	err = os.WriteFile(path, content, 0600)
	if err != nil {
		return err
	}
	return nil
}

// Parse provided address and return either an IP address as a one
// element list or a DNS name as one element list. The arrays are
// returned as then it is easy to pass these returned elements to the
// functions that generates CSR (Certificate Signing Request).
func resolveAddr(addr string) ([]net.IP, []string) {
	ipAddr := net.ParseIP(addr)
	if ipAddr != nil {
		return []net.IP{ipAddr}, []string{}
	}
	return []net.IP{}, []string{addr}
}

// Generate or regenerate agent key and CSR (Certificate Signing
// Request). They are generated when they do not exist. They are
// regenerated only if regenCerts is true. If they exist and
// regenCerts is false then they are used.
func generateCerts(agentAddr string, regenCerts bool) ([]byte, string, error) {
	_, err := os.Stat(KeyPEMFile)
	if err != nil && os.IsNotExist(err) {
		regenCerts = true
	}

	agentIPs, agentNames := resolveAddr(agentAddr)

	var agentToken []byte
	var csrPEM []byte
	var privKeyPEM []byte
	if regenCerts {
		// generate private key and CSR
		var fingerprint [32]byte
		privKeyPEM, csrPEM, fingerprint, err = pki.GenKeyAndCSR("agent", agentNames, agentIPs)
		if err != nil {
			return nil, "", err
		}
		agentToken = fingerprint[:]

		// save private key to file
		err = writeAgentFile(KeyPEMFile, privKeyPEM)
		if err != nil {
			return nil, "", err
		}

		err = writeAgentFile(AgentTokenFile, agentToken)
		if err != nil {
			log.Errorf("problem with storing agent token in %s: %s", AgentTokenFile, err)
			return nil, "", err
		}

		log.Printf("agent token stored in %s", AgentTokenFile)
		log.Printf("agent key, agent token and CSR (re)generated")
	} else {
		// generate CSR using existing private key and agent token
		privKeyPEM, err = os.ReadFile(KeyPEMFile)
		if err != nil {
			return nil, "", errors.Wrapf(err, "could not load key PEM file: %s", KeyPEMFile)
		}

		agentToken, err = os.ReadFile(AgentTokenFile)
		if err != nil {
			msg := "could not load agent token from file: %s, try to force the certs regeneration"
			return nil, "", errors.Wrapf(err, msg, AgentTokenFile)
		}

		csrPEM, _, err = pki.GenCSRUsingKey("agent", agentNames, agentIPs, privKeyPEM)
		if err != nil {
			return nil, "", err
		}
		log.Printf("loaded existing agent key and generated CSR")
	}

	// convert fingerprint to hex string
	fingerprintStr := storkutil.BytesToHex(agentToken)

	return csrPEM, fingerprintStr, nil
}

// Prepare agent registration request payload to Stork Server in JSON format.
func prepareRegistrationRequestPayload(csrPEM []byte, serverToken, agentToken, agentAddr string, agentPort int) (*bytes.Buffer, error) {
	values := map[string]interface{}{
		"address":     agentAddr,
		"agentPort":   agentPort,
		"agentCSR":    string(csrPEM),
		"serverToken": serverToken,
		"agentToken":  agentToken,
	}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot marshal registration request")
	}
	return bytes.NewBuffer(jsonValue), nil
}

// Register agent in Stork Server under provided URL using reqPayload in request.
// If retry is true then registration is repeated until it connection to server
// is established. This case is used when agent automatically tries to register
// during startup.
// If the agent is already registered then only ID is returned, the certificates are empty.
func registerAgentInServer(client *http.Client, baseSrvURL *url.URL, reqPayload *bytes.Buffer, retry bool) (int64, string, string, error) {
	url, _ := baseSrvURL.Parse("api/machines")
	var err error
	var resp *http.Response
	for {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url.String(), reqPayload)
		if err != nil {
			return 0, "", "", errors.Wrapf(err, "problem with preparing registering request")
		}
		req.Header.Add("Content-Type", "application/json")
		resp, err = client.Do(req)
		if err == nil {
			break
		}

		// If connection is refused and retries are enabled than wait for 10 seconds
		// and try again. This method is used in case of agent token based registration
		// to allow smooth automated registration even if server is down for some time.
		// In case of server token based registration this method is invoked manually so
		// it should fail immediately if there is no connection to the server.
		if retry && strings.Contains(err.Error(), "connection refused") {
			log.Println("sleeping for 10 seconds before next registration attempt")
			time.Sleep(10 * time.Second)
		} else {
			return 0, "", "", errors.Wrapf(err, "problem with registering machine")
		}
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return 0, "", "", errors.Wrapf(err, "problem with reading server's response while registering the machine")
	}

	// Special case - the agent is already registered
	if resp.StatusCode == http.StatusConflict {
		location := resp.Header.Get("Location")
		lastSeparatorIdx := strings.LastIndex(location, "/")
		if lastSeparatorIdx < 0 || lastSeparatorIdx+1 >= len(location) {
			return 0, "", "", errors.New("missing machine ID in response from server for registration request")
		}
		machineID, err := strconv.Atoi(location[lastSeparatorIdx+1:])
		if err != nil {
			return 0, "", "", errors.New("bad machine ID in response from server for registration request")
		}
		return int64(machineID), "", "", nil
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return 0, "", "", errors.Wrapf(err, "problem with parsing server's response while registering the machine: %v", result)
	}
	errTxt := result["error"]
	if errTxt != nil {
		msg := "problem with registering machine"
		errTxtStr, ok := errTxt.(string)
		if ok {
			msg = fmt.Sprintf("problem with registering machine: %s", errTxtStr)
		}
		return 0, "", "", errors.New(msg)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		errTxt = result["message"]
		var msg string
		if errTxt != nil {
			errTxtStr, ok := errTxt.(string)
			if ok {
				msg = fmt.Sprintf("problem with registering machine: %s", errTxtStr)
			} else {
				msg = "problem with registering machine"
			}
		} else {
			msg = fmt.Sprintf("problem with registering machine: http status code %d", resp.StatusCode)
		}
		return 0, "", "", errors.New(msg)
	}

	// check received machine ID
	if result["id"] == nil {
		return 0, "", "", errors.New("missing ID in response from server for registration request")
	}
	machineID, ok := result["id"].(float64)
	if !ok {
		return 0, "", "", errors.New("bad ID in response from server for registration request")
	}
	// check received serverCACert
	if result["serverCACert"] == nil {
		return 0, "", "", errors.New("missing serverCACert in response from server for registration request")
	}
	serverCACert, ok := result["serverCACert"].(string)
	if !ok {
		return 0, "", "", errors.New("bad serverCACert in response from server for registration request")
	}
	// check received agentCert
	if result["agentCert"] == nil {
		return 0, "", "", errors.New("missing agentCert in response from server for registration request")
	}
	agentCert, ok := result["agentCert"].(string)
	if !ok {
		return 0, "", "", errors.New("bad agentCert in response from server for registration request")
	}
	// all ok
	log.Printf("machine registered")
	return int64(machineID), serverCACert, agentCert, nil
}

// Check certs received from server.
func checkAndStoreCerts(serverCACert, agentCert string) error {
	// check certs
	_, err := pki.ParseCert([]byte(serverCACert))
	if err != nil {
		return errors.Wrapf(err, "cannot parse server CA cert")
	}
	_, err = pki.ParseCert([]byte(agentCert))
	if err != nil {
		return errors.Wrapf(err, "cannot parse agent cert")
	}

	// save certs
	err = writeAgentFile(CertPEMFile, []byte(agentCert))
	if err != nil {
		return errors.Wrapf(err, "cannot write agent cert")
	}
	err = writeAgentFile(RootCAFile, []byte(serverCACert))
	if err != nil {
		return errors.Wrapf(err, "cannot write server CA cert")
	}
	log.Printf("stored agent signed cert and CA cert")
	return nil
}

// Ping Stork Agent service via Stork Server. It is used during manual registration
// to confirm that TLS connection between agent and server can be established.
func pingAgentViaServer(client *http.Client, baseSrvURL *url.URL, machineID int64, serverToken, agentToken string) error {
	urlSuffix := fmt.Sprintf("api/machines/%d/ping", machineID)
	url, err := baseSrvURL.Parse(urlSuffix)
	if err != nil {
		return errors.Wrapf(err, "problem with preparing url %s + %s", baseSrvURL.String(), urlSuffix)
	}
	req := map[string]interface{}{
		"serverToken": serverToken,
		"agentToken":  agentToken,
	}
	jsonReq, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url.String(), bytes.NewBuffer(jsonReq))
	if err != nil {
		return errors.Wrapf(err, "problem with preparing http request")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return errors.Wrapf(err, "problem with pinging machine")
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return errors.Wrapf(err, "problem with reading server's response while pinging machine")
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	// normally the response is empty so unmarshalling is failing, if it didn't fail it means
	// that there could be some error information
	if err == nil {
		errTxt := result["error"]
		if errTxt != nil {
			msg := "problem with pinging machine"
			errTxtStr, ok := errTxt.(string)
			if ok {
				msg = fmt.Sprintf("problem with pinging machine: %s", errTxtStr)
			}
			return errors.New(msg)
		}
	}
	if resp.StatusCode >= http.StatusBadRequest {
		var msg string
		if result != nil {
			errTxt := result["message"]
			if errTxt != nil {
				errTxtStr, ok := errTxt.(string)
				if ok {
					msg = fmt.Sprintf("problem with pinging machine: %s", errTxtStr)
				}
			}
		}
		if msg == "" {
			msg = fmt.Sprintf("problem with pinging machine: http status code %d", resp.StatusCode)
		}
		return errors.New(msg)
	}

	log.Printf("machine ping over TLS: OK")

	return nil
}

// Main function used to register an agent (with a given address and
// port) in a server indicated by given URL. If regenCerts is true
// then agent key and cert are regenerated, otherwise the ones stored
// in files are used. RegenCerts is used when registration is run
// manually. If retry is true then registration is retried if
// connection to server cannot be established. This case is used when
// registration is automatic during agent service startup. Server
// token can be provided in manual registration via command line
// switch. This way the agent will be immediately authorized in the
// server. If server token is empty (in automatic registration or
// when it is not provided in manual registration) then agent is added
// to server but requires manual authorization in web UI.
func Register(serverURL, serverToken, agentAddr, agentPort string, regenCerts bool, retry bool) bool {
	// parse URL to server
	baseSrvURL, err := url.Parse(serverURL)
	if err != nil || baseSrvURL.String() == "" {
		log.Errorf("cannot parse server URL: %s: %s", serverURL, err)
		return false
	}

	// Get server token from user (if not provided in cmd line) to authenticate in the server.
	// Do not ask if regenCerts is true (ie. Register is called from agent).
	serverToken2 := serverToken
	if serverToken == "" && regenCerts {
		serverToken2, err = promptUserForServerToken()
		if err != nil {
			log.Errorf("problem with getting server token: %s", err)
			return false
		}
	}

	agentAddr, agentPortInt, err := getAgentAddrAndPortFromUser(agentAddr, agentPort)
	if err != nil {
		return false
	}

	// Generate agent private key and cert. If they already exist then regenerate them if forced.
	csrPEM, agentToken, err := generateCerts(agentAddr, regenCerts)
	if err != nil {
		log.Errorf("problem with generating certs: %s", err)
		return false
	}

	// Use cert fingerprint as agent token.
	// Agent token is another mode for checking identity of an agent.
	log.Println("=============================================================================")
	log.Printf("AGENT TOKEN: %s", agentToken)
	log.Println("=============================================================================")

	if serverToken2 == "" {
		log.Println("authorize the machine in the Stork web UI")
	} else {
		log.Println("machine will be automatically registered using the server token")
		log.Println("agent token is printed above for informational purposes only")
		log.Println("the user does not need to copy or verify the agent token during the registration using the server token")
		log.Println("it will be sent to the server but it is not directly used in this type of the machine registration")
	}

	// prepare http client to connect to Stork Server
	client := &http.Client{}

	// register new machine i.e. current agent
	reqPayload, err := prepareRegistrationRequestPayload(csrPEM, serverToken2, agentToken, agentAddr, agentPortInt)
	if err != nil {
		log.Errorln(err.Error())
		return false
	}
	log.Println("try to register agent in Stork Server")
	machineID, serverCACert, agentCert, err := registerAgentInServer(client, baseSrvURL, reqPayload, retry)
	if err != nil {
		log.Errorln(err.Error())
		return false
	}

	// store certs
	// if server and agent CA certs are empty then the agent should use existing ones
	if serverCACert != "" && agentCert != "" {
		err = checkAndStoreCerts(serverCACert, agentCert)
		if err != nil {
			log.Errorf("problem with certs: %s", err)
			return false
		}
	}

	if serverToken2 != "" {
		// invoke getting machine state via server
		for i := 1; i < 4; i++ {
			err = pingAgentViaServer(client, baseSrvURL, machineID, serverToken2, agentToken)
			if err == nil {
				break
			}
			if i < 3 {
				log.Errorf("retrying ping %d/3 due to error: %s", i, err)
				time.Sleep(2 * time.Duration(i) * time.Second)
			}
		}
		if err != nil {
			log.Errorf("cannot ping machine: %s", err)
			return false
		}
	}

	return true
}
