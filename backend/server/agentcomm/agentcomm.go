package agentcomm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/security/advancedtls"

	agentapi "isc.org/stork/api"
	keactrl "isc.org/stork/appctrl/kea"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

// Settings specific to communication with Agents.
type AgentsSettings struct {
}

// Holds runtime communication statistics with Kea daemons via
// a given agent.
type AgentKeaCommStats struct {
	CurrentErrorsCA      int64            // generall errors in communication to/via CA
	CurrentErrorsDaemons map[string]int64 // errors returned by particular daemon (including CA)
}

// Holds runtime communication statistics with Bind9 daemon for
// a given agent.
type AgentBind9CommStats struct {
	CurrentErrorsRNDC  int64
	CurrentErrorsStats int64
}

type AppCommStatsKey struct {
	Address string
	Port    int64
}

// Holds runtime statistics of communication with a given agent and
// with the apps behind this agent.
type AgentStats struct {
	CurrentErrors int64
	AppCommStats  map[AppCommStatsKey]interface{}
	mutex         *sync.Mutex
}

// Runtime information about the agent, e.g. connection, communication
// statistics.
type Agent struct {
	Address  string
	Client   agentapi.AgentClient
	GrpcConn *grpc.ClientConn
	Stats    AgentStats
}

// Prepare TLS credentials with configured certs and verification options.
func prepareTLSCreds(caCertPEM, serverCertPEM, serverKeyPEM []byte) (credentials.TransportCredentials, error) {
	// Load the certificates from disk
	certificate, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		return nil, errors.Wrapf(err, "could not load client key pair")
	}

	// Create a certificate pool from the certificate authority
	certPool := x509.NewCertPool()

	// Append the client certificates from the CA
	if ok := certPool.AppendCertsFromPEM(caCertPEM); !ok {
		return nil, errors.New("failed to append ca certs")
	}

	// Prepare structure for advanced TLS with custom agent verification.
	// It sets root CA cert and stork server cert. It also defines hook function
	// that checks if stork agent cert is still valid.
	options := &advancedtls.ClientOptions{
		// set root CA cert to validate agent's certificate
		RootOptions: advancedtls.RootCertificateOptions{
			RootCACerts: certPool,
		},
		// set TLS client (stork server) cert to present its identity to server (stork agent)
		IdentityOptions: advancedtls.IdentityCertificateOptions{
			Certificates: []tls.Certificate{certificate},
		},
		// check cert and if it matches host IP
		VType: advancedtls.CertAndHostVerification,
		// additional verification hook function that checks if stork agent is not using old cert
		// VerifyPeer: func(params *advancedtls.VerificationFuncParams) (*advancedtls.VerificationResults, error) {
		// 	// TODO: add here check if agent cert is not old (compare it with
		// 	// CertFingerprint from Machine db record)
		// 	return &advancedtls.VerificationResults{}, nil
		// },
	}
	creds, err := advancedtls.NewClientCreds(options)
	if err != nil {
		log.Fatalf("advancedtls.NewClientCreds(%v) failed: %v", options, err)
	}

	return creds, nil
}

// Prepare gRPC connection to agent.
func (agent *Agent) MakeGrpcConnection(caCertPEM, serverCertPEM, serverKeyPEM []byte) error {
	// If there is any old connection then clean it up
	if agent.GrpcConn != nil {
		agent.GrpcConn.Close()
	}

	// Prepare TLS credentials
	creds, err := prepareTLSCreds(caCertPEM, serverCertPEM, serverKeyPEM)
	if err != nil {
		return errors.WithMessagef(err, "problem with preparing TLS credentials")
	}

	// Setup new connection
	grpcConn, err := grpc.Dial(agent.Address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return errors.Wrapf(err, "problem with dial to agent %s", agent.Address)
	}

	agent.Client = agentapi.NewAgentClient(grpcConn)
	agent.GrpcConn = grpcConn

	return nil
}

// Interface for interacting with Agents via gRPC.
type ConnectedAgents interface {
	Shutdown()
	GetConnectedAgent(address string) (*Agent, error)
	GetConnectedAgentStats(adddress string, port int64) *AgentStats
	Ping(ctx context.Context, address string, agentPort int64) error
	GetState(ctx context.Context, address string, agentPort int64) (*State, error)
	ForwardRndcCommand(ctx context.Context, dbApp *dbmodel.App, command string) (*RndcOutput, error)
	ForwardToNamedStats(ctx context.Context, agentAddress string, agentPort int64, statsAddress string, statsPort int64, path string, statsOutput interface{}) error
	ForwardToKeaOverHTTP(ctx context.Context, dbApp *dbmodel.App, commands []*keactrl.Command, cmdResponses ...interface{}) (*KeaCmdsResult, error)
	TailTextFile(ctx context.Context, agentAddress string, agentPort int64, path string, offset int64) ([]string, error)
}

// Agents management map. It tracks Agents currently connected to the Server.
type connectedAgentsData struct {
	Settings      *AgentsSettings
	EventCenter   eventcenter.EventCenter
	AgentsMap     map[string]*Agent
	CommLoopReqs  chan *commLoopReq
	DoneCommLoop  chan bool
	Wg            *sync.WaitGroup
	serverCertPEM []byte
	serverKeyPEM  []byte
	caCertPEM     []byte
}

// Create new ConnectedAgents objects.
func NewConnectedAgents(settings *AgentsSettings, eventCenter eventcenter.EventCenter, caCertPEM, serverCertPEM, serverKeyPEM []byte) ConnectedAgents {
	agents := connectedAgentsData{
		Settings:      settings,
		EventCenter:   eventCenter,
		AgentsMap:     make(map[string]*Agent),
		CommLoopReqs:  make(chan *commLoopReq),
		DoneCommLoop:  make(chan bool),
		Wg:            &sync.WaitGroup{},
		caCertPEM:     caCertPEM,
		serverCertPEM: serverCertPEM,
		serverKeyPEM:  serverKeyPEM,
	}

	agents.Wg.Add(1)
	go agents.communicationLoop()

	return &agents
}

// Shutdown agents in agents map.
func (agents *connectedAgentsData) Shutdown() {
	log.Printf("Stopping communication with agents")
	for _, agent := range agents.AgentsMap {
		agent.GrpcConn.Close()
	}

	close(agents.CommLoopReqs)
	agents.DoneCommLoop <- true
	agents.Wg.Wait()
	log.Printf("Stopped communication with agents")
}

// Get Agent object by its address.
func (agents *connectedAgentsData) GetConnectedAgent(address string) (*Agent, error) {
	// Look for agent in Agents map and if found then return it
	agent, ok := agents.AgentsMap[address]
	if ok {
		log.WithFields(log.Fields{
			"address": address,
		}).Info("connecting to existing agent")
		return agent, nil
	}

	// Agent not found so allocate agent and prepare connection
	agent = new(Agent)
	agent.Address = address
	agent.Stats.AppCommStats = make(map[AppCommStatsKey]interface{})
	agent.Stats.mutex = new(sync.Mutex)
	err := agent.MakeGrpcConnection(agents.caCertPEM, agents.serverCertPEM, agents.serverKeyPEM)
	if err != nil {
		return nil, err
	}

	// Store it in Agents map
	agents.AgentsMap[address] = agent
	log.WithFields(log.Fields{
		"address": address,
	}).Info("connecting to new agent")

	return agent, nil
}

// Returns statistics for the connected agent. The statistics include number
// of errors to communicate with the agent and the number of errors to
// communicate with the apps behind the agent.
func (agents *connectedAgentsData) GetConnectedAgentStats(address string, port int64) *AgentStats {
	if port != 0 {
		address = fmt.Sprintf("%s:%d", address, port)
	}
	if agent, ok := agents.AgentsMap[address]; ok {
		return &agent.Stats
	}
	return nil
}
