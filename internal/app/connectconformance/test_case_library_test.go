// Copyright 2023 The Connect Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package connectconformance

import (
	"sort"
	"testing"

	"connectrpc.com/conformance/internal/app/connectconformance/testsuites"
	conformancev1 "connectrpc.com/conformance/internal/gen/proto/go/connectrpc/conformance/v1"
	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestNewTestCaseLibrary(t *testing.T) {
	t.Parallel()

	testData := map[string]string{
		"basic.yaml": `
                    name: Basic
                    testCases:
                      - request:
                            testName: basic-unary
                            streamType: STREAM_TYPE_UNARY
                      - request:
                            testName: basic-client-stream
                            streamType: STREAM_TYPE_CLIENT_STREAM
                      - request:
                            testName: basic-server-stream
                            streamType: STREAM_TYPE_SERVER_STREAM
                      - request:
                            testName: basic-bidi-stream
                            streamType: STREAM_TYPE_FULL_DUPLEX_BIDI_STREAM`,
		"tls.yaml": `
                    name: TLS
                    reliesOnTls: true
                    testCases:
                      - request:
                            testName: tls-unary
                            streamType: STREAM_TYPE_UNARY
                      - request:
                            testName: tls-client-stream
                            streamType: STREAM_TYPE_CLIENT_STREAM
                      - request:
                            testName: tls-server-stream
                            streamType: STREAM_TYPE_SERVER_STREAM
                      - request:
                            testName: tls-bidi-stream
                            streamType: STREAM_TYPE_FULL_DUPLEX_BIDI_STREAM`,
		"tls-client-certs.yaml": `
                    name: TLS Client Certs
                    reliesOnTls: true
                    reliesOnTlsClientCerts: true
                    testCases:
                      - request:
                            testName: tls-client-cert-unary
                            streamType: STREAM_TYPE_UNARY`,
		"connect-get.yaml": `
                    name: Connect GET
                    relevantProtocols: [PROTOCOL_CONNECT]
                    reliesOnConnectGet: true
                    testCases:
                      - request:
                            testName: connect-get-unary
                            streamType: STREAM_TYPE_UNARY`,
		"connect-version-client-required.yaml": `
                    name: Connect Version Required (client)
                    mode: TEST_MODE_CLIENT
                    relevantProtocols: [PROTOCOL_CONNECT]
                    connectVersionMode: CONNECT_VERSION_MODE_REQUIRE
                    testCases:
                      - request:
                            testName: unary-without-connect-version-header
                            streamType: STREAM_TYPE_UNARY`,
		"connect-version-server-required.yaml": `
                    name: Connect Version Required (server)
                    mode: TEST_MODE_SERVER
                    relevantProtocols: [PROTOCOL_CONNECT]
                    connectVersionMode: CONNECT_VERSION_MODE_REQUIRE
                    testCases:
                      - request:
                            testName: unary-without-connect-version-header
                            streamType: STREAM_TYPE_UNARY`,
		"connect-version-client-not-required.yaml": `
                    name: Connect Version Optional (client)
                    mode: TEST_MODE_CLIENT
                    relevantProtocols: [PROTOCOL_CONNECT]
                    connectVersionMode: CONNECT_VERSION_MODE_IGNORE
                    testCases:
                      - request:
                            testName: unary-without-connect-version-header
                            streamType: STREAM_TYPE_UNARY`,
		"connect-version-server-not-required.yaml": `
                    name: Connect Version Optional (server)
                    mode: TEST_MODE_SERVER
                    relevantProtocols: [PROTOCOL_CONNECT]
                    connectVersionMode: CONNECT_VERSION_MODE_IGNORE
                    testCases:
                      - request:
                            testName: unary-without-connect-version-header
                            streamType: STREAM_TYPE_UNARY`,
		"max-receive-limit": `
                    name: Max Receive Size (server)
                    mode: TEST_MODE_SERVER
                    reliesOnMessageReceiveLimit: true
                    testCases:
                      - request:
                            testName: unary-exceeds-limit
                            streamType: STREAM_TYPE_UNARY`,
	}
	testSuiteData := make(map[string][]byte, len(testData))
	for k, v := range testData {
		testSuiteData[k] = []byte(v)
	}
	testSuites, err := parseTestSuites(testSuiteData)
	require.NoError(t, err)

	// there is some repetition, but we want them to be able to
	// vary and evolve independently, so we won't consolidate
	//nolint:dupl
	testCases := []struct {
		name   string
		config []configCase
		mode   conformancev1.TestSuite_TestMode
		cases  map[serverInstance][]string
	}{
		{
			name: "client mode",
			config: []configCase{
				{
					Version:     conformancev1.HTTPVersion_HTTP_VERSION_1,
					Protocol:    conformancev1.Protocol_PROTOCOL_CONNECT,
					Codec:       conformancev1.Codec_CODEC_PROTO,
					Compression: conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:  conformancev1.StreamType_STREAM_TYPE_UNARY,
				},
				{
					Version:     conformancev1.HTTPVersion_HTTP_VERSION_1,
					Protocol:    conformancev1.Protocol_PROTOCOL_CONNECT,
					Codec:       conformancev1.Codec_CODEC_PROTO,
					Compression: conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:  conformancev1.StreamType_STREAM_TYPE_UNARY,
					UseTLS:      true,
				},
				{
					Version:           conformancev1.HTTPVersion_HTTP_VERSION_1,
					Protocol:          conformancev1.Protocol_PROTOCOL_CONNECT,
					Codec:             conformancev1.Codec_CODEC_PROTO,
					Compression:       conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:        conformancev1.StreamType_STREAM_TYPE_UNARY,
					UseTLS:            true,
					UseTLSClientCerts: true,
				},
				{
					Version:       conformancev1.HTTPVersion_HTTP_VERSION_1,
					Protocol:      conformancev1.Protocol_PROTOCOL_CONNECT,
					Codec:         conformancev1.Codec_CODEC_PROTO,
					Compression:   conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:    conformancev1.StreamType_STREAM_TYPE_UNARY,
					UseConnectGET: true,
				},
				{
					Version:            conformancev1.HTTPVersion_HTTP_VERSION_1,
					Protocol:           conformancev1.Protocol_PROTOCOL_CONNECT,
					Codec:              conformancev1.Codec_CODEC_PROTO,
					Compression:        conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:         conformancev1.StreamType_STREAM_TYPE_UNARY,
					ConnectVersionMode: conformancev1.TestSuite_CONNECT_VERSION_MODE_REQUIRE,
				},
				{
					Version:                conformancev1.HTTPVersion_HTTP_VERSION_1,
					Protocol:               conformancev1.Protocol_PROTOCOL_CONNECT,
					Codec:                  conformancev1.Codec_CODEC_PROTO,
					Compression:            conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:             conformancev1.StreamType_STREAM_TYPE_UNARY,
					UseMessageReceiveLimit: true,
				},
				{
					Version:     conformancev1.HTTPVersion_HTTP_VERSION_2,
					Protocol:    conformancev1.Protocol_PROTOCOL_GRPC,
					Codec:       conformancev1.Codec_CODEC_PROTO,
					Compression: conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:  conformancev1.StreamType_STREAM_TYPE_FULL_DUPLEX_BIDI_STREAM,
				},
			},
			mode: conformancev1.TestSuite_TEST_MODE_CLIENT,
			cases: map[serverInstance][]string{
				{
					protocol:    conformancev1.Protocol_PROTOCOL_CONNECT,
					httpVersion: conformancev1.HTTPVersion_HTTP_VERSION_1,
					useTLS:      false,
				}: {
					"Basic/HTTPVersion:1/Protocol:PROTOCOL_CONNECT/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/basic-unary",
					"Connect GET/HTTPVersion:1/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/connect-get-unary",
					"Connect Version Required (client)/HTTPVersion:1/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/unary-without-connect-version-header",
				},
				{
					protocol:    conformancev1.Protocol_PROTOCOL_CONNECT,
					httpVersion: conformancev1.HTTPVersion_HTTP_VERSION_1,
					useTLS:      true,
				}: {
					"TLS/HTTPVersion:1/Protocol:PROTOCOL_CONNECT/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/tls-unary",
				},
				{
					protocol:          conformancev1.Protocol_PROTOCOL_CONNECT,
					httpVersion:       conformancev1.HTTPVersion_HTTP_VERSION_1,
					useTLS:            true,
					useTLSClientCerts: true,
				}: {
					"TLS Client Certs/HTTPVersion:1/Protocol:PROTOCOL_CONNECT/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/tls-client-cert-unary",
				},
				{
					protocol:    conformancev1.Protocol_PROTOCOL_GRPC,
					httpVersion: conformancev1.HTTPVersion_HTTP_VERSION_2,
					useTLS:      false,
				}: {
					"Basic/HTTPVersion:2/Protocol:PROTOCOL_GRPC/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/basic-bidi-stream",
				},
			},
		},

		{
			name: "server mode",
			config: []configCase{
				{
					Version:     conformancev1.HTTPVersion_HTTP_VERSION_1,
					Protocol:    conformancev1.Protocol_PROTOCOL_CONNECT,
					Codec:       conformancev1.Codec_CODEC_PROTO,
					Compression: conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:  conformancev1.StreamType_STREAM_TYPE_UNARY,
				},
				{
					Version:     conformancev1.HTTPVersion_HTTP_VERSION_1,
					Protocol:    conformancev1.Protocol_PROTOCOL_CONNECT,
					Codec:       conformancev1.Codec_CODEC_PROTO,
					Compression: conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:  conformancev1.StreamType_STREAM_TYPE_UNARY,
					UseTLS:      true,
				},
				{
					Version:           conformancev1.HTTPVersion_HTTP_VERSION_1,
					Protocol:          conformancev1.Protocol_PROTOCOL_CONNECT,
					Codec:             conformancev1.Codec_CODEC_PROTO,
					Compression:       conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:        conformancev1.StreamType_STREAM_TYPE_UNARY,
					UseTLS:            true,
					UseTLSClientCerts: true,
				},
				{
					Version:       conformancev1.HTTPVersion_HTTP_VERSION_1,
					Protocol:      conformancev1.Protocol_PROTOCOL_CONNECT,
					Codec:         conformancev1.Codec_CODEC_PROTO,
					Compression:   conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:    conformancev1.StreamType_STREAM_TYPE_UNARY,
					UseConnectGET: true,
				},
				{
					Version:            conformancev1.HTTPVersion_HTTP_VERSION_1,
					Protocol:           conformancev1.Protocol_PROTOCOL_CONNECT,
					Codec:              conformancev1.Codec_CODEC_PROTO,
					Compression:        conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:         conformancev1.StreamType_STREAM_TYPE_UNARY,
					ConnectVersionMode: conformancev1.TestSuite_CONNECT_VERSION_MODE_IGNORE,
				},
				{
					Version:                conformancev1.HTTPVersion_HTTP_VERSION_1,
					Protocol:               conformancev1.Protocol_PROTOCOL_CONNECT,
					Codec:                  conformancev1.Codec_CODEC_PROTO,
					Compression:            conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:             conformancev1.StreamType_STREAM_TYPE_UNARY,
					UseMessageReceiveLimit: true,
				},
				{
					Version:     conformancev1.HTTPVersion_HTTP_VERSION_2,
					Protocol:    conformancev1.Protocol_PROTOCOL_GRPC,
					Codec:       conformancev1.Codec_CODEC_PROTO,
					Compression: conformancev1.Compression_COMPRESSION_IDENTITY,
					StreamType:  conformancev1.StreamType_STREAM_TYPE_FULL_DUPLEX_BIDI_STREAM,
				},
			},
			mode: conformancev1.TestSuite_TEST_MODE_SERVER,
			cases: map[serverInstance][]string{
				{
					protocol:    conformancev1.Protocol_PROTOCOL_CONNECT,
					httpVersion: conformancev1.HTTPVersion_HTTP_VERSION_1,
					useTLS:      false,
				}: {
					"Basic/HTTPVersion:1/Protocol:PROTOCOL_CONNECT/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/basic-unary",
					"Connect GET/HTTPVersion:1/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/connect-get-unary",
					"Connect Version Optional (server)/HTTPVersion:1/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/unary-without-connect-version-header",
					"Max Receive Size (server)/HTTPVersion:1/Protocol:PROTOCOL_CONNECT/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/unary-exceeds-limit",
				},
				{
					protocol:    conformancev1.Protocol_PROTOCOL_CONNECT,
					httpVersion: conformancev1.HTTPVersion_HTTP_VERSION_1,
					useTLS:      true,
				}: {
					"TLS/HTTPVersion:1/Protocol:PROTOCOL_CONNECT/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/tls-unary",
				},
				{
					protocol:          conformancev1.Protocol_PROTOCOL_CONNECT,
					httpVersion:       conformancev1.HTTPVersion_HTTP_VERSION_1,
					useTLS:            true,
					useTLSClientCerts: true,
				}: {
					"TLS Client Certs/HTTPVersion:1/Protocol:PROTOCOL_CONNECT/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/tls-client-cert-unary",
				},
				{
					protocol:    conformancev1.Protocol_PROTOCOL_GRPC,
					httpVersion: conformancev1.HTTPVersion_HTTP_VERSION_2,
					useTLS:      false,
				}: {
					"Basic/HTTPVersion:2/Protocol:PROTOCOL_GRPC/Codec:CODEC_PROTO/Compression:COMPRESSION_IDENTITY/basic-bidi-stream",
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCaseLib, err := newTestCaseLibrary(testSuites, testCase.config, testCase.mode)
			require.NoError(t, err)
			results := make(map[serverInstance][]string, len(testCaseLib.casesByServer))
			for svrKey, testCaseProtos := range testCaseLib.casesByServer {
				names := make([]string, len(testCaseProtos))
				for i, testCaseProto := range testCaseProtos {
					names[i] = testCaseProto.Request.TestName
				}
				sort.Strings(names)
				results[svrKey] = names
			}
			require.Empty(t, cmp.Diff(testCase.cases, results), "- wanted; + got")
		})
	}
}

func TestParseTestSuites_EmbeddedTestSuites(t *testing.T) {
	t.Parallel()
	testSuiteData, err := testsuites.LoadTestSuites()
	require.NoError(t, err)
	allSuites, err := parseTestSuites(testSuiteData)
	require.NoError(t, err)
	_ = allSuites
	// TODO: basic assertions about the embedded test suites?
}

func TestExpandRequestData(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		testCaseJSON string
		expectErr    string
		expectSizes  []int
	}{
		{
			name: "unary-no-expand",
			testCaseJSON: `{
				"request": {
					"requestMessages":[
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.UnaryRequest",
							"request_data": "abcdefgh"
						}
					]
				},
				"expandRequests":[]
			}`,
			/* negative means size is unchanged */
			expectSizes: []int{-1},
		},
		{
			name: "unary-expand",
			testCaseJSON: `{
				"request": {
					"requestMessages":[
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.UnaryRequest",
							"request_data": "abcdefgh"
						}
					]
				},
				"expandRequests":[{"size_relative_to_limit":123}]
			}`,
			expectSizes: []int{200*1024 + 123},
		},
		{
			name: "server-stream-no-expand",
			testCaseJSON: `{
				"request": {
					"requestMessages":[
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.ServerStreamRequest",
							"request_data": "abcdefgh"
						}
					]
				},
				"expandRequests":[]
			}`,
			expectSizes: []int{-1},
		},
		{
			name: "server-stream-expand",
			testCaseJSON: `{
				"request": {
					"requestMessages":[
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.ServerStreamRequest",
							"request_data": "abcdefgh"
						}
					]
				},
				"expandRequests":[{"size_relative_to_limit":123}]
			}`,
			expectSizes: []int{200*1024 + 123},
		},
		{
			name: "client-stream-no-expand",
			testCaseJSON: `{
				"request": {
					"requestMessages":[
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.ClientStreamRequest",
							"request_data": "abcdefgh"
						},
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.ClientStreamRequest",
							"request_data": "abcdefgh"
						},
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.ClientStreamRequest",
							"request_data": "abcdefgh"
						}
					]
				},
				"expandRequests":[]
			}`,
			expectSizes: []int{-1, -1, -1},
		},
		{
			name: "client-stream-expand",
			testCaseJSON: `{
				"request": {
					"requestMessages":[
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.ClientStreamRequest",
							"request_data": "abcdefgh"
						},
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.ClientStreamRequest",
							"request_data": "abcdefgh"
						},
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.ClientStreamRequest",
							"request_data": "abcdefgh"
						},
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.ClientStreamRequest",
							"request_data": "abcdefgh"
						}
					]
				},
				"expandRequests":[
					{"size_relative_to_limit":123},
					{"size_relative_to_limit":null},
					{"size_relative_to_limit":-123}
				]
			}`,
			expectSizes: []int{200*1024 + 123, -1, 200*1024 - 123, -1},
		},
		{
			name: "bidi-stream-no-expand",
			testCaseJSON: `{
				"request": {
					"requestMessages":[
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.BidiStreamRequest",
							"request_data": "abcdefgh"
						},
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.BidiStreamRequest",
							"request_data": "abcdefgh"
						}
					]
				},
				"expandRequests":[]
			}`,
			expectSizes: []int{-1, -1},
		},
		{
			name: "bidi-stream-expand",
			testCaseJSON: `{
				"request": {
					"requestMessages":[
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.BidiStreamRequest",
							"request_data": "abcdefgh"
						},
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.BidiStreamRequest",
							"request_data": "abcdefgh"
						},
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.BidiStreamRequest",
							"request_data": "abcdefgh"
						}
					]
				},
				"expandRequests":[
					{"size_relative_to_limit":null},
					{"size_relative_to_limit":null},
					{"size_relative_to_limit":0}
				]
			}`,
			expectSizes: []int{-1, -1, 200 * 1024},
		},
		{
			name: "too-many-expand-directives",
			testCaseJSON: `{
				"request": {
					"requestMessages":[]
				},
				"expandRequests":[
					{"size_relative_to_limit":null},
					{"size_relative_to_limit":123}
				]
			}`,
			expectErr: "expand directives indicate 2 messages, but there are only 0 requests",
		},
		{
			name: "invalid-adjustment",
			testCaseJSON: `{
				"request": {
					"requestMessages":[
						{
							"@type": "type.googleapis.com/connectrpc.conformance.v1.BidiStreamRequest",
							"request_data": "abcdefgh"
						}
					]
				},
				"expandRequests":[
					{"size_relative_to_limit":-300000}
				]
			}`,
			expectErr: "expand directive #1 (-300000) results in an invalid request size: -95200",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var testCaseProto conformancev1.TestCase
			err := protojson.Unmarshal([]byte(testCase.testCaseJSON), &testCaseProto)
			require.NoError(t, err)
			reqs := testCaseProto.Request.RequestMessages
			initialSizes := make([]int, len(reqs))
			for i, req := range reqs {
				initialSizes[i] = len(req.Value)
			}
			err = expandRequestData(&testCaseProto)
			if testCase.expectErr != "" {
				require.ErrorContains(t, err, testCase.expectErr)
				return
			}
			require.NoError(t, err)
			for i, req := range reqs {
				expectedSize := testCase.expectSizes[i]
				if expectedSize < 0 {
					expectedSize = initialSizes[i]
				}
				require.Len(t, req.Value, expectedSize)
			}
		})
	}
}

func TestPopulateExpectedResponse(t *testing.T) {
	t.Parallel()

	requestHeaders := []*conformancev1.Header{
		{
			Name:  "reqHeader",
			Value: []string{"reqHeaderVal"},
		},
	}
	responseHeaders := []*conformancev1.Header{
		{
			Name:  "fooHeader",
			Value: []string{"fooHeaderVal"},
		},
		{
			Name:  "barHeader",
			Value: []string{"barHeaderVal1", "barHeaderVal2"},
		},
	}
	responseTrailers := []*conformancev1.Header{
		{
			Name:  "fooTrailer",
			Value: []string{"fooTrailerVal"},
		},
		{
			Name:  "barTrailer",
			Value: []string{"barTrailerVal1", "barTrailerVal2"},
		},
	}
	data1 := []byte("data1")
	data2 := []byte("data2")

	headerAny, err := anypb.New(&conformancev1.Header{
		Name:  "detail test",
		Value: []string{"val1", "val2"},
	})
	require.NoError(t, err)

	errorDef := &conformancev1.Error{
		Code:    int32(connect.CodeResourceExhausted),
		Message: proto.String("all resources exhausted"),
	}

	errorDetailsDef := &conformancev1.Error{
		Code:    errorDef.Code,
		Message: errorDef.Message,
		Details: []*anypb.Any{headerAny},
	}

	// Unary Requests
	unarySuccessReq, err := anypb.New(&conformancev1.UnaryRequest{
		ResponseDefinition: &conformancev1.UnaryResponseDefinition{
			ResponseHeaders: responseHeaders,
			Response: &conformancev1.UnaryResponseDefinition_ResponseData{
				ResponseData: data1,
			},
			ResponseTrailers: responseTrailers,
		},
	})
	require.NoError(t, err)

	unaryErrorReq, err := anypb.New(&conformancev1.UnaryRequest{
		ResponseDefinition: &conformancev1.UnaryResponseDefinition{
			ResponseHeaders: responseHeaders,
			Response: &conformancev1.UnaryResponseDefinition_Error{
				Error: errorDef,
			},
			ResponseTrailers: responseTrailers,
		},
	})
	require.NoError(t, err)

	unaryErrorDetailsReq, err := anypb.New(&conformancev1.UnaryRequest{
		ResponseDefinition: &conformancev1.UnaryResponseDefinition{
			ResponseHeaders: responseHeaders,
			Response: &conformancev1.UnaryResponseDefinition_Error{
				Error: errorDetailsDef,
			},
			ResponseTrailers: responseTrailers,
		},
	})
	require.NoError(t, err)

	unaryNoResponseReq, err := anypb.New(&conformancev1.UnaryRequest{
		ResponseDefinition: &conformancev1.UnaryResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseTrailers: responseTrailers,
		},
	})
	require.NoError(t, err)

	unaryNoDefReq, err := anypb.New(&conformancev1.UnaryRequest{})
	require.NoError(t, err)

	// Client Stream Requests
	clientStreamSuccessReq, err := anypb.New(&conformancev1.ClientStreamRequest{
		ResponseDefinition: &conformancev1.UnaryResponseDefinition{
			ResponseHeaders: responseHeaders,
			Response: &conformancev1.UnaryResponseDefinition_ResponseData{
				ResponseData: data1,
			},
			ResponseTrailers: responseTrailers,
		},
		RequestData: data1,
	})
	require.NoError(t, err)

	clientStreamErrorReq, err := anypb.New(&conformancev1.ClientStreamRequest{
		ResponseDefinition: &conformancev1.UnaryResponseDefinition{
			ResponseHeaders: responseHeaders,
			Response: &conformancev1.UnaryResponseDefinition_Error{
				Error: errorDef,
			},
			ResponseTrailers: responseTrailers,
		},
	})
	require.NoError(t, err)

	clientStreamErrorDetailsReq, err := anypb.New(&conformancev1.ClientStreamRequest{
		ResponseDefinition: &conformancev1.UnaryResponseDefinition{
			ResponseHeaders: responseHeaders,
			Response: &conformancev1.UnaryResponseDefinition_Error{
				Error: errorDetailsDef,
			},
			ResponseTrailers: responseTrailers,
		},
	})
	require.NoError(t, err)

	clientStreamNoResponseReq, err := anypb.New(&conformancev1.ClientStreamRequest{
		ResponseDefinition: &conformancev1.UnaryResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseTrailers: responseTrailers,
		},
		RequestData: data1,
	})
	require.NoError(t, err)

	clientStreamNoDefReq, err := anypb.New(&conformancev1.ClientStreamRequest{
		RequestData: data1,
	})
	require.NoError(t, err)

	clientStreamReq2, err := anypb.New(&conformancev1.ClientStreamRequest{
		RequestData: data1,
	})
	require.NoError(t, err)

	// Server Stream Requests
	serverStreamSuccessReq, err := anypb.New(&conformancev1.ServerStreamRequest{
		ResponseDefinition: &conformancev1.StreamResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseData:     [][]byte{data1, data2},
			ResponseDelayMs:  1000,
			ResponseTrailers: responseTrailers,
		},
	})
	require.NoError(t, err)

	serverStreamErrorReq, err := anypb.New(&conformancev1.ServerStreamRequest{
		ResponseDefinition: &conformancev1.StreamResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseData:     [][]byte{data1, data2},
			ResponseDelayMs:  1000,
			Error:            errorDef,
			ResponseTrailers: responseTrailers,
		},
	})
	require.NoError(t, err)

	serverStreamNoResponseReq, err := anypb.New(&conformancev1.ServerStreamRequest{
		ResponseDefinition: &conformancev1.StreamResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseDelayMs:  1000,
			ResponseTrailers: responseTrailers,
		},
	})
	require.NoError(t, err)

	serverStreamNoResponseWithErrorReq, err := anypb.New(&conformancev1.ServerStreamRequest{
		ResponseDefinition: &conformancev1.StreamResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseDelayMs:  1000,
			Error:            errorDef,
			ResponseTrailers: responseTrailers,
		},
	})
	require.NoError(t, err)

	serverStreamNoDefReq, err := anypb.New(&conformancev1.ServerStreamRequest{})
	require.NoError(t, err)

	// Bidi Stream Requests
	bidiStreamHalfDuplexSuccessReq, err := anypb.New(&conformancev1.BidiStreamRequest{
		ResponseDefinition: &conformancev1.StreamResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseData:     [][]byte{data1, data2},
			ResponseDelayMs:  1000,
			ResponseTrailers: responseTrailers,
		},
		RequestData: data1,
	})
	require.NoError(t, err)

	bidiStreamReq2, err := anypb.New(&conformancev1.BidiStreamRequest{
		RequestData: data2,
	})
	require.NoError(t, err)

	bidiStreamHalfDuplexErrorReq, err := anypb.New(&conformancev1.BidiStreamRequest{
		ResponseDefinition: &conformancev1.StreamResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseData:     [][]byte{data1, data2},
			ResponseDelayMs:  1000,
			Error:            errorDef,
			ResponseTrailers: responseTrailers,
		},
		RequestData: data1,
	})
	require.NoError(t, err)

	bidiStreamHalfDuplexNoResponseReq, err := anypb.New(&conformancev1.BidiStreamRequest{
		ResponseDefinition: &conformancev1.StreamResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseDelayMs:  1000,
			ResponseTrailers: responseTrailers,
		},
		RequestData: data1,
	})
	require.NoError(t, err)

	bidiStreamHalfDuplexNoResponseWithErrorReq, err := anypb.New(&conformancev1.BidiStreamRequest{
		ResponseDefinition: &conformancev1.StreamResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseDelayMs:  1000,
			Error:            errorDef,
			ResponseTrailers: responseTrailers,
		},
	})
	require.NoError(t, err)

	bidiStreamHalfDuplexNoDefReq, err := anypb.New(&conformancev1.BidiStreamRequest{
		RequestData: data1,
	})
	require.NoError(t, err)

	bidiStreamFullDuplexSuccessReq, err := anypb.New(&conformancev1.BidiStreamRequest{
		ResponseDefinition: &conformancev1.StreamResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseData:     [][]byte{data1, data2},
			ResponseDelayMs:  1000,
			ResponseTrailers: responseTrailers,
		},
		RequestData: data1,
		FullDuplex:  true,
	})
	require.NoError(t, err)

	bidiStreamFullDuplexErrorReq, err := anypb.New(&conformancev1.BidiStreamRequest{
		ResponseDefinition: &conformancev1.StreamResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseData:     [][]byte{data1, data2},
			ResponseDelayMs:  1000,
			Error:            errorDef,
			ResponseTrailers: responseTrailers,
		},
		RequestData: data1,
		FullDuplex:  true,
	})
	require.NoError(t, err)

	bidiStreamFullDuplexNoResponseWithErrorReq, err := anypb.New(&conformancev1.BidiStreamRequest{
		ResponseDefinition: &conformancev1.StreamResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseDelayMs:  1000,
			Error:            errorDef,
			ResponseTrailers: responseTrailers,
		},
		RequestData: data1,
		FullDuplex:  true,
	})
	require.NoError(t, err)

	bidiStreamFullDuplexNoResponseReq, err := anypb.New(&conformancev1.BidiStreamRequest{
		ResponseDefinition: &conformancev1.StreamResponseDefinition{
			ResponseHeaders:  responseHeaders,
			ResponseDelayMs:  1000,
			ResponseTrailers: responseTrailers,
		},
		RequestData: data1,
		FullDuplex:  true,
	})
	require.NoError(t, err)

	bidiStreamFullDuplexNoDefReq, err := anypb.New(&conformancev1.BidiStreamRequest{
		RequestData: data1,
		FullDuplex:  true,
	})
	require.NoError(t, err)

	// Request Infos
	unaryErrorReqInfo, err := anypb.New(&conformancev1.ConformancePayload_RequestInfo{
		RequestHeaders: requestHeaders,
		Requests:       []*anypb.Any{unaryErrorReq},
	})
	require.NoError(t, err)

	unaryErrorDetailsReqInfo, err := anypb.New(&conformancev1.ConformancePayload_RequestInfo{
		RequestHeaders: requestHeaders,
		Requests:       []*anypb.Any{unaryErrorDetailsReq},
	})
	require.NoError(t, err)

	clientStreamErrorReqInfo, err := anypb.New(&conformancev1.ConformancePayload_RequestInfo{
		RequestHeaders: requestHeaders,
		Requests:       []*anypb.Any{clientStreamErrorReq, clientStreamReq2},
	})
	require.NoError(t, err)

	clientStreamErrorDetailsReqInfo, err := anypb.New(&conformancev1.ConformancePayload_RequestInfo{
		RequestHeaders: requestHeaders,
		Requests:       []*anypb.Any{clientStreamErrorDetailsReq, clientStreamReq2},
	})
	require.NoError(t, err)

	serverStreamErrorReqInfo, err := anypb.New(&conformancev1.ConformancePayload_RequestInfo{
		RequestHeaders: requestHeaders,
		Requests:       []*anypb.Any{serverStreamNoResponseWithErrorReq},
	})
	require.NoError(t, err)

	bidiStreamHalfDuplexNoResponseWithErrorReqInfo, err := anypb.New(&conformancev1.ConformancePayload_RequestInfo{
		RequestHeaders: requestHeaders,
		Requests:       []*anypb.Any{bidiStreamHalfDuplexNoResponseWithErrorReq, bidiStreamReq2},
	})
	require.NoError(t, err)

	bidiStreamFullDuplexNoResponseWithErrorReqInfo, err := anypb.New(&conformancev1.ConformancePayload_RequestInfo{
		RequestHeaders: requestHeaders,
		Requests:       []*anypb.Any{bidiStreamFullDuplexNoResponseWithErrorReq, bidiStreamReq2},
	})
	require.NoError(t, err)

	testCases := []struct {
		testName   string
		request    *conformancev1.ClientCompatRequest
		expected   *conformancev1.ClientResponseResult
		requireErr bool
	}{
		{
			testName: "unary success",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_UNARY,
				RequestMessages: []*anypb.Any{unarySuccessReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Payloads: []*conformancev1.ConformancePayload{
					{
						Data: data1,
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							RequestHeaders: requestHeaders,
							Requests:       []*anypb.Any{unarySuccessReq},
						},
					},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "unary error",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_UNARY,
				RequestMessages: []*anypb.Any{unaryErrorReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Error: &conformancev1.Error{
					Code:    errorDef.Code,
					Message: errorDef.Message,
					Details: []*anypb.Any{unaryErrorReqInfo},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "unary error specifying details appends req info to details",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_UNARY,
				RequestMessages: []*anypb.Any{unaryErrorDetailsReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Error: &conformancev1.Error{
					Code:    errorDef.Code,
					Message: errorDef.Message,
					Details: []*anypb.Any{headerAny, unaryErrorDetailsReqInfo},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "unary no response set",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_UNARY,
				RequestMessages: []*anypb.Any{unaryNoResponseReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Payloads: []*conformancev1.ConformancePayload{
					{
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							RequestHeaders: requestHeaders,
							Requests:       []*anypb.Any{unaryNoResponseReq},
						},
					},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "unary no definition set",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_UNARY,
				RequestMessages: []*anypb.Any{unaryNoDefReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				Payloads: []*conformancev1.ConformancePayload{
					{
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							RequestHeaders: requestHeaders,
							Requests:       []*anypb.Any{unaryNoDefReq},
						},
					},
				},
			},
		},
		{
			testName: "client stream success",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_CLIENT_STREAM,
				RequestMessages: []*anypb.Any{clientStreamSuccessReq, clientStreamReq2},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Payloads: []*conformancev1.ConformancePayload{
					{
						Data: data1,
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							RequestHeaders: requestHeaders,
							Requests:       []*anypb.Any{clientStreamSuccessReq, clientStreamReq2},
						},
					},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "client stream error",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_CLIENT_STREAM,
				RequestMessages: []*anypb.Any{clientStreamErrorReq, clientStreamReq2},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Error: &conformancev1.Error{
					Code:    errorDef.Code,
					Message: errorDef.Message,
					Details: []*anypb.Any{clientStreamErrorReqInfo},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "client stream error specifying details appends req info to details",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_CLIENT_STREAM,
				RequestMessages: []*anypb.Any{clientStreamErrorDetailsReq, clientStreamReq2},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Error: &conformancev1.Error{
					Code:    errorDetailsDef.Code,
					Message: errorDetailsDef.Message,
					Details: []*anypb.Any{headerAny, clientStreamErrorDetailsReqInfo},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "client stream no response set",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_CLIENT_STREAM,
				RequestMessages: []*anypb.Any{clientStreamNoResponseReq, clientStreamReq2},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Payloads: []*conformancev1.ConformancePayload{
					{
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							RequestHeaders: requestHeaders,
							Requests:       []*anypb.Any{clientStreamNoResponseReq, clientStreamReq2},
						},
					},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "client stream no definition set",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_CLIENT_STREAM,
				RequestMessages: []*anypb.Any{clientStreamNoDefReq, clientStreamReq2},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				Payloads: []*conformancev1.ConformancePayload{
					{
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							RequestHeaders: requestHeaders,
							Requests:       []*anypb.Any{clientStreamNoDefReq, clientStreamReq2},
						},
					},
				},
			},
		},
		{
			testName: "server stream success",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_SERVER_STREAM,
				RequestMessages: []*anypb.Any{serverStreamSuccessReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Payloads: []*conformancev1.ConformancePayload{
					{
						Data: data1,
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							RequestHeaders: requestHeaders,
							Requests:       []*anypb.Any{serverStreamSuccessReq},
						},
					},
					{
						Data: data2,
					},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "server stream error with responses",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_SERVER_STREAM,
				RequestMessages: []*anypb.Any{serverStreamErrorReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Payloads: []*conformancev1.ConformancePayload{
					{
						Data: data1,
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							RequestHeaders: requestHeaders,
							Requests:       []*anypb.Any{serverStreamErrorReq},
						},
					},
					{
						Data: data2,
					},
				},
				Error:            errorDef,
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "server stream error with no responses",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_SERVER_STREAM,
				RequestMessages: []*anypb.Any{serverStreamNoResponseWithErrorReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Error: &conformancev1.Error{
					Code:    errorDef.Code,
					Message: errorDef.Message,
					Details: []*anypb.Any{serverStreamErrorReqInfo},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "server stream no response set",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_SERVER_STREAM,
				RequestMessages: []*anypb.Any{serverStreamNoResponseReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders:  responseHeaders,
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "server stream no definition set",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_SERVER_STREAM,
				RequestMessages: []*anypb.Any{serverStreamNoDefReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{},
		},
		{
			testName: "half duplex bidi stream success",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_HALF_DUPLEX_BIDI_STREAM,
				RequestMessages: []*anypb.Any{bidiStreamHalfDuplexSuccessReq, bidiStreamReq2},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Payloads: []*conformancev1.ConformancePayload{
					{
						Data: data1,
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							RequestHeaders: requestHeaders,
							Requests:       []*anypb.Any{bidiStreamHalfDuplexSuccessReq, bidiStreamReq2},
						},
					},
					{
						Data: data2,
					},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "half duplex bidi stream error with responses",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_HALF_DUPLEX_BIDI_STREAM,
				RequestMessages: []*anypb.Any{bidiStreamHalfDuplexErrorReq, bidiStreamReq2},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Payloads: []*conformancev1.ConformancePayload{
					{
						Data: data1,
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							RequestHeaders: requestHeaders,
							Requests:       []*anypb.Any{bidiStreamHalfDuplexErrorReq, bidiStreamReq2},
						},
					},
					{
						Data: data2,
					},
				},
				Error:            errorDef,
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "half duplex bidi stream error with no responses",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_HALF_DUPLEX_BIDI_STREAM,
				RequestMessages: []*anypb.Any{bidiStreamHalfDuplexNoResponseWithErrorReq, bidiStreamReq2},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Error: &conformancev1.Error{
					Code:    errorDef.Code,
					Message: errorDef.Message,
					Details: []*anypb.Any{bidiStreamHalfDuplexNoResponseWithErrorReqInfo},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "half duplex bidi stream no response set",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_HALF_DUPLEX_BIDI_STREAM,
				RequestMessages: []*anypb.Any{bidiStreamHalfDuplexNoResponseReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders:  responseHeaders,
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "half duplex bidi stream no definition set",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_HALF_DUPLEX_BIDI_STREAM,
				RequestMessages: []*anypb.Any{bidiStreamHalfDuplexNoDefReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{},
		},
		{
			testName: "full duplex bidi stream success",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_FULL_DUPLEX_BIDI_STREAM,
				RequestMessages: []*anypb.Any{bidiStreamFullDuplexSuccessReq, bidiStreamReq2},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Payloads: []*conformancev1.ConformancePayload{
					{
						Data: data1,
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							RequestHeaders: requestHeaders,
							Requests:       []*anypb.Any{bidiStreamFullDuplexSuccessReq},
						},
					},
					{
						Data: data2,
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							Requests: []*anypb.Any{bidiStreamReq2},
						},
					},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "full duplex bidi stream error with responses",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_FULL_DUPLEX_BIDI_STREAM,
				RequestMessages: []*anypb.Any{bidiStreamFullDuplexErrorReq, bidiStreamReq2},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Payloads: []*conformancev1.ConformancePayload{
					{
						Data: data1,
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							RequestHeaders: requestHeaders,
							Requests:       []*anypb.Any{bidiStreamFullDuplexErrorReq},
						},
					},
					{
						Data: data2,
						RequestInfo: &conformancev1.ConformancePayload_RequestInfo{
							Requests: []*anypb.Any{bidiStreamReq2},
						},
					},
				},
				Error:            errorDef,
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "full duplex bidi stream error with no responses",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_FULL_DUPLEX_BIDI_STREAM,
				RequestMessages: []*anypb.Any{bidiStreamFullDuplexNoResponseWithErrorReq, bidiStreamReq2},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders: responseHeaders,
				Error: &conformancev1.Error{
					Code:    errorDef.Code,
					Message: errorDef.Message,
					Details: []*anypb.Any{bidiStreamFullDuplexNoResponseWithErrorReqInfo},
				},
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "full duplex bidi stream no response set",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_FULL_DUPLEX_BIDI_STREAM,
				RequestMessages: []*anypb.Any{bidiStreamFullDuplexNoResponseReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{
				ResponseHeaders:  responseHeaders,
				ResponseTrailers: responseTrailers,
			},
		},
		{
			testName: "full duplex bidi stream no definition set",
			request: &conformancev1.ClientCompatRequest{
				StreamType:      conformancev1.StreamType_STREAM_TYPE_FULL_DUPLEX_BIDI_STREAM,
				RequestMessages: []*anypb.Any{bidiStreamFullDuplexNoDefReq},
				RequestHeaders:  requestHeaders,
			},
			expected: &conformancev1.ClientResponseResult{},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.testName, func(t *testing.T) {
			t.Parallel()

			tc := &conformancev1.TestCase{ //nolint:varnamelen
				Request: testCase.request,
			}
			err := populateExpectedResponse(tc)
			require.NoError(t, err)
			if testCase.requireErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				assert.Empty(t, cmp.Diff(testCase.expected, tc.ExpectedResponse, protocmp.Transform()))
			}
		})
	}
}