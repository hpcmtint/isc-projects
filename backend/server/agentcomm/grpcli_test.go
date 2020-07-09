package agentcomm

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	agentapi "isc.org/stork/api"
	dbmodel "isc.org/stork/server/database/model"
	storktest "isc.org/stork/server/test"
)

// makeAccessPoint is an utility to make single element app access point slice.
func makeAccessPoint(tp, address, key string, port int64) (ap []*agentapi.AccessPoint) {
	return append(ap, &agentapi.AccessPoint{
		Type:    tp,
		Address: address,
		Port:    port,
		Key:     key,
	})
}

// Setup function for the unit tests. It creates a fake agent running at
// 127.0.0.1:8080. The returned function performs a test teardown and
// should be invoked when the unit test finishes.
func setupGrpcliTestCase(t *testing.T) (*MockAgentClient, ConnectedAgents, func()) {
	settings := AgentsSettings{}
	fec := &storktest.FakeEventCenter{}
	agents := NewConnectedAgents(&settings, fec)

	// pre-add an agent
	addr := "127.0.0.1:8080"
	agent, err := agents.GetConnectedAgent(addr)
	require.NoError(t, err)

	// create mock AgentClient and patch agent to point to it
	ctrl := gomock.NewController(t)
	mockAgentClient := NewMockAgentClient(ctrl)
	agent.Client = mockAgentClient

	return mockAgentClient, agents, func() {
		ctrl.Finish()
	}
}

//go:generate mockgen -package=agentcomm -destination=api_mock.go isc.org/stork/api AgentClient

func TestGetState(t *testing.T) {
	mockAgentClient, agents, teardown := setupGrpcliTestCase(t)
	defer teardown()

	// Call GetState
	expVer := "123"
	rsp := agentapi.GetStateRsp{
		AgentVersion: expVer,
		Apps: []*agentapi.App{
			{
				Type:         AppTypeKea,
				AccessPoints: makeAccessPoint(AccessPointControl, "1.2.3.4", "", 1234),
			},
		},
	}
	mockAgentClient.EXPECT().GetState(gomock.Any(), gomock.Any()).
		Return(&rsp, nil)

	// Check response
	ctx := context.Background()
	state, err := agents.GetState(ctx, "127.0.0.1", 8080)
	require.NoError(t, err)
	require.Equal(t, expVer, state.AgentVersion)
	require.Equal(t, AppTypeKea, state.Apps[0].Type)
}

// Test that a command can be successfully forwarded to Kea and the response
// can be parsed.
func TestForwardToKeaOverHTTP(t *testing.T) {
	mockAgentClient, agents, teardown := setupGrpcliTestCase(t)
	defer teardown()

	rsp := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{{
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: `[
            {
                "result": 1,
                "text": "operation failed"
            },
            {
                "result": 0,
                "text": "operation succeeded",
                "arguments": {
                    "success": true
                }
            }
        ]`}},
	}

	mockAgentClient.EXPECT().ForwardToKeaOverHTTP(gomock.Any(), gomock.Any()).
		Return(&rsp, nil)

	ctx := context.Background()
	daemons, _ := NewKeaDaemons("dhcp4", "dhcp6")
	command, _ := NewKeaCommand("test-command", daemons, nil)
	actualResponse := KeaResponseList{}
	dbApp := &dbmodel.App{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []*KeaCommand{command}, &actualResponse)
	require.NoError(t, err)
	require.NotNil(t, actualResponse)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	require.NoError(t, cmdsResult.CmdsErrors[0])

	responseList := actualResponse
	require.Len(t, responseList, 2)

	require.Equal(t, 1, responseList[0].Result)
	require.Equal(t, "operation failed", responseList[0].Text)
	require.Nil(t, responseList[0].Arguments)

	require.Equal(t, 0, responseList[1].Result)
	require.Equal(t, "operation succeeded", responseList[1].Text)
	require.NotNil(t, responseList[1].Arguments)
	require.Len(t, *responseList[1].Arguments, 1)
	require.Contains(t, *responseList[1].Arguments, "success")

	agent, err := agents.GetConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.Stats.CurrentErrors)

	appCommStats, ok := agent.Stats.AppCommStats[AppCommStatsKey{"localhost", 8000}].(*AgentKeaCommStats)
	require.True(t, ok)

	require.Zero(t, appCommStats.CurrentErrorsCA)
	require.Contains(t, appCommStats.CurrentErrorsDaemons, "dhcp4")
	require.Contains(t, appCommStats.CurrentErrorsDaemons, "dhcp6")

	require.EqualValues(t, 1, appCommStats.CurrentErrorsDaemons["dhcp4"])
	require.Zero(t, appCommStats.CurrentErrorsDaemons["dhcp6"])
}

// Test that two commands can be successfully forwarded to Kea and the response
// can be parsed.
func TestForwardToKeaOverHTTPWith2Cmds(t *testing.T) {
	mockAgentClient, agents, teardown := setupGrpcliTestCase(t)
	defer teardown()

	rsp := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{{
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: `[
            {
                "result": 1,
                "text": "operation failed"
            },
            {
                "result": 0,
                "text": "operation succeeded",
                "arguments": {
                    "success": true
                }
            }
        ]`}, {
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: `[
            {
                "result": 1,
                "text": "operation failed"
            }
        ]`}},
	}

	mockAgentClient.EXPECT().ForwardToKeaOverHTTP(gomock.Any(), gomock.Any()).
		Return(&rsp, nil)

	ctx := context.Background()
	daemons, _ := NewKeaDaemons("dhcp4", "dhcp6")
	command1, _ := NewKeaCommand("test-command", daemons, nil)
	command2, _ := NewKeaCommand("test-command", daemons, nil)
	actualResponse1 := KeaResponseList{}
	actualResponse2 := KeaResponseList{}
	dbApp := &dbmodel.App{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []*KeaCommand{command1, command2}, &actualResponse1, &actualResponse2)
	require.NoError(t, err)
	require.NotNil(t, actualResponse1)
	require.NotNil(t, actualResponse2)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 2)
	require.NoError(t, cmdsResult.CmdsErrors[0])
	require.NoError(t, cmdsResult.CmdsErrors[1])

	responseList := actualResponse1
	require.Len(t, responseList, 2)

	require.Equal(t, 1, responseList[0].Result)
	require.Equal(t, "operation failed", responseList[0].Text)
	require.Nil(t, responseList[0].Arguments)

	require.Equal(t, 0, responseList[1].Result)
	require.Equal(t, "operation succeeded", responseList[1].Text)
	require.NotNil(t, responseList[1].Arguments)
	require.Len(t, *responseList[1].Arguments, 1)
	require.Contains(t, *responseList[1].Arguments, "success")

	responseList = actualResponse2
	require.Len(t, responseList, 1)

	require.Equal(t, 1, responseList[0].Result)
	require.Equal(t, "operation failed", responseList[0].Text)
	require.Nil(t, responseList[0].Arguments)

	agent, err := agents.GetConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.Stats.CurrentErrors)

	appCommStats, ok := agent.Stats.AppCommStats[AppCommStatsKey{"localhost", 8000}].(*AgentKeaCommStats)
	require.True(t, ok)

	require.Zero(t, appCommStats.CurrentErrorsCA)
	require.Contains(t, appCommStats.CurrentErrorsDaemons, "dhcp4")
	require.Contains(t, appCommStats.CurrentErrorsDaemons, "dhcp6")

	require.EqualValues(t, 2, appCommStats.CurrentErrorsDaemons["dhcp4"])
	require.Zero(t, appCommStats.CurrentErrorsDaemons["dhcp6"])
}

// Test that the error is returned when the response to the forwarded Kea command
// is malformed.
func TestForwardToKeaOverHTTPInvalidResponse(t *testing.T) {
	mockAgentClient, agents, teardown := setupGrpcliTestCase(t)
	defer teardown()

	rsp := agentapi.ForwardToKeaOverHTTPRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		KeaResponses: []*agentapi.KeaResponse{{
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: `[
            {
                "result": "a string"
            }
        ]`}},
	}
	mockAgentClient.EXPECT().ForwardToKeaOverHTTP(gomock.Any(), gomock.Any()).
		Return(&rsp, nil)

	ctx := context.Background()
	command, _ := NewKeaCommand("test-command", nil, nil)
	actualResponse := KeaResponseList{}
	dbApp := &dbmodel.App{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8000,
			Key:     "",
		}},
	}
	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []*KeaCommand{command}, &actualResponse)
	require.NoError(t, err)
	require.NotNil(t, cmdsResult)
	require.NoError(t, cmdsResult.Error)
	require.Len(t, cmdsResult.CmdsErrors, 1)
	// and now for our command we get an error
	require.Error(t, cmdsResult.CmdsErrors[0])

	agent, err := agents.GetConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.Stats.CurrentErrors)

	appCommStats, ok := agent.Stats.AppCommStats[AppCommStatsKey{"localhost", 8000}].(*AgentKeaCommStats)
	require.True(t, ok)

	require.EqualValues(t, 1, appCommStats.CurrentErrorsCA)
}

// Test that a statistics request can be successfully forwarded to named
// statistics-channel and the output can be parsed.
func TestForwardToNamedStats(t *testing.T) {
	mockAgentClient, agents, teardown := setupGrpcliTestCase(t)
	defer teardown()

	rsp := agentapi.ForwardToNamedStatsRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		NamedStatsResponse: &agentapi.NamedStatsResponse{
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: `{
                             "json-stats-version": "1.2.",
                             "views": {
                                 "_default": {
                                     "resolver": {
                                         "cachestats": {
                                             "CacheHits": 11,
                                             "CacheMisses": 12
                                         }
                                     }
                                 }
                             }
                        }`,
		},
	}

	mockAgentClient.EXPECT().ForwardToNamedStats(gomock.Any(), gomock.Any()).
		Return(&rsp, nil)

	ctx := context.Background()
	actualResponse := NamedStatsGetResponse{}
	err := agents.ForwardToNamedStats(ctx, "127.0.0.1", 8080, "localhost", 8000, "", &actualResponse)
	require.NoError(t, err)
	require.NotNil(t, actualResponse)
	require.Len(t, *actualResponse.Views, 1)
	require.Contains(t, *actualResponse.Views, "_default")

	agent, err := agents.GetConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.Stats.CurrentErrors)

	appCommStats, ok := agent.Stats.AppCommStats[AppCommStatsKey{"localhost", 8000}].(*AgentBind9CommStats)
	require.True(t, ok)
	require.Zero(t, appCommStats.CurrentErrorsStats)
}

// Test that the error is returned when the response to the forwarded
// named statistics request is malformed.
func TestForwardToNamedStatsInvalidResponse(t *testing.T) {
	mockAgentClient, agents, teardown := setupGrpcliTestCase(t)
	defer teardown()

	rsp := agentapi.ForwardToNamedStatsRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		NamedStatsResponse: &agentapi.NamedStatsResponse{
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: `{
                          "views": "not the views you are looking for",
            }`},
	}
	mockAgentClient.EXPECT().ForwardToNamedStats(gomock.Any(), gomock.Any()).
		Return(&rsp, nil)

	ctx := context.Background()
	actualResponse := NamedStatsGetResponse{}
	err := agents.ForwardToNamedStats(ctx, "127.0.0.1", 8080, "localhost", 8000, "", &actualResponse)
	require.Error(t, err)

	agent, err := agents.GetConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.Stats.CurrentErrors)

	appCommStats, ok := agent.Stats.AppCommStats[AppCommStatsKey{"localhost", 8000}].(*AgentBind9CommStats)
	require.True(t, ok)
	require.EqualValues(t, 1, appCommStats.CurrentErrorsStats)
}

// Test that a command can be successfully forwarded to rndc and the response
// can be parsed.
func TestForwardRndcCommand(t *testing.T) {
	mockAgentClient, agents, teardown := setupGrpcliTestCase(t)
	defer teardown()

	rsp := agentapi.ForwardRndcCommandRsp{
		Status: &agentapi.Status{
			Code: 0,
		},
		RndcResponse: &agentapi.RndcResponse{
			Status: &agentapi.Status{
				Code: 0,
			},
			Response: "all good",
		},
	}

	mockAgentClient.EXPECT().ForwardRndcCommand(gomock.Any(), gomock.Any()).
		Return(&rsp, nil)

	ctx := context.Background()
	dbApp := &dbmodel.App{
		Machine: &dbmodel.Machine{
			Address:   "127.0.0.1",
			AgentPort: 8080,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "127.0.0.1",
			Port:    953,
			Key:     "",
		}},
	}

	out, err := agents.ForwardRndcCommand(ctx, dbApp, "test")
	require.NoError(t, err)
	require.Equal(t, out.Output, "all good")
	require.NoError(t, out.Error)

	agent, err := agents.GetConnectedAgent("127.0.0.1:8080")
	require.NoError(t, err)
	require.NotNil(t, agent)
	require.Zero(t, agent.Stats.CurrentErrors)

	appCommStats, ok := agent.Stats.AppCommStats[AppCommStatsKey{"127.0.0.1", 953}].(*AgentBind9CommStats)
	require.True(t, ok)

	require.Zero(t, appCommStats.CurrentErrorsRNDC)
}
