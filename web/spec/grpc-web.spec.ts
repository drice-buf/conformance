// This is from the grpc-web interop interop_client.js file
// https://github.com/grpc/grpc-web/blob/master/test/interop/interop_client.js

/**
 *
 * Copyright 2018 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

import {TestServiceClient, UnimplementedServiceClient} from "../gen/proto/grpc-web/grpc/testing/TestServiceClientPb";
import {Empty} from "../gen/proto/grpc-web/grpc/testing/empty_pb";
import {
    EchoStatus,
    Payload,
    ResponseParameters,
    SimpleRequest,
    StreamingOutputCallRequest
} from "../gen/proto/grpc-web/grpc/testing/messages_pb";
import caseless = require("caseless");

function multiDone(done: DoneFn, count: number) {
    return function() {
        count -= 1;
        if (count <= 0) {
            done();
        }
    };
}

describe("grpc_web", function () {
    const host = __karma__.config.host;
    const port = __karma__.config.port;
    const SERVER_HOST = `https://${host}:${port}`;
    const client = new TestServiceClient(SERVER_HOST, null, null);
    it("empty_unary", function (done) {
        client.emptyCall(new Empty(), null, (err, response) => {
            expect(err).toBeNull();
            expect(response).toEqual(new Empty());
            done();
        });
    });
    it("empty_unary_with_timeout", function (done) {
        const deadlineMs = 1000; // 1 second
        client.emptyCall(new Empty(), { deadline: `${Date.now() + deadlineMs}` }, (err, response) => {
            expect(err).toBeNull();
            expect(response).toEqual(new Empty());
            done();
        });
    });
    it("large_unary", function (done) {
        const req = new SimpleRequest();
        const size = 314159;

        const payload = new Payload();
        payload.setBody('0'.repeat(271828));

        req.setPayload(payload);
        req.setResponseSize(size);
        client.unaryCall(req, null, (err, response) => {
            expect(err).toBeNull();
            expect(response.getPayload()).toBeDefined();
            expect(response.getPayload()?.getBody().length).toEqual(size);
            done();
        });
    });
    it("server_stream", function (done) {
        const sizes = [31415, 9, 2653, 58979];
        const doneFn = multiDone(done, sizes.length)

        const responseParams = sizes.map((size, idx) => {
            const param = new ResponseParameters();
            param.setSize(size);
            param.setIntervalUs(idx * 10);
            return param;
        });

        const req = new StreamingOutputCallRequest();
        req.setResponseParametersList(responseParams);

        const stream = client.streamingOutputCall(req);
        let responseCount = 0;
        stream.on('data', (response) => {
            expect(response.getPayload()).toBeDefined();
            expect(response.getPayload()?.getBody().length).toEqual(sizes[responseCount]);
            responseCount++;
            doneFn();
        });
    });
    it("custom_metadata", function (done) {
        const doneFn = multiDone(done, 3)
        const size = 314159;
        const ECHO_INITIAL_KEY = "x-grpc-test-echo-initial";
        const ECHO_INITIAL_VALUE = "test_initial_metadata_value";
        const ECHO_TRAILING_KEY = "x-grpc-test-echo-trailing-bin";
        const ECHO_TRAILING_VALUE = 0xababab;

        const payload = new Payload();
        payload.setBody('0'.repeat(271828));

        const req = new SimpleRequest();
        req.setPayload(payload);
        req.setResponseSize(size);

        const call = client.unaryCall(req, {
            [ECHO_INITIAL_KEY]: ECHO_INITIAL_VALUE,
            [ECHO_TRAILING_KEY]: ECHO_TRAILING_VALUE.toString()
        }, (err, response) => {
            expect(response.getPayload()).toBeDefined();
            expect(response.getPayload()?.getBody().length).toEqual(size);
            doneFn();
        });

        call.on('metadata', (metadata) => {
            expect(metadata).toBeDefined()
            const m = caseless(metadata); // http header is case-insensitive
            expect(m.has(ECHO_INITIAL_KEY) != false).toBeTrue();
            expect(m.get(ECHO_INITIAL_KEY)).toEqual(ECHO_INITIAL_VALUE.toString());
            doneFn();
        });

        call.on('status', (status) => {
            expect(status.metadata).toBeDefined()
            const m = caseless(status.metadata); // http header is case-insensitive
            expect(m.has(ECHO_TRAILING_KEY) != false).toBeTrue();
            expect(m.get(ECHO_TRAILING_KEY)).toEqual(ECHO_TRAILING_VALUE.toString());
            doneFn();
        });
    })
    it("status_code_and_message", function (done) {
        const req = new SimpleRequest();

        const TEST_STATUS_MESSAGE = 'test status message';
        const echoStatus = new EchoStatus();
        echoStatus.setCode(2);
        echoStatus.setMessage(TEST_STATUS_MESSAGE);

        req.setResponseStatus(echoStatus);

        client.unaryCall(req, null, (err) => {
            expect(err).toBeDefined();
            expect('code' in err).toBeTrue();
            expect('message' in err).toBeTrue();
            expect(err.code).toEqual(2);
            expect(err.message).toEqual(TEST_STATUS_MESSAGE);
            done();
        });
    });
    it("special_status", function (done) {
        const req = new SimpleRequest();

        const TEST_STATUS_MESSAGE = `\t\ntest with whitespace\r\nand Unicode BMP ☺ and non-BMP 😈\t\n`;
        const echoStatus = new EchoStatus();
        echoStatus.setCode(2);
        echoStatus.setMessage(TEST_STATUS_MESSAGE);

        req.setResponseStatus(echoStatus);

        client.unaryCall(req, null, (err) => {
            expect(err).toBeDefined();
            expect('code' in err).toBeTrue();
            expect('message' in err).toBeTrue();
            expect(err.code).toEqual(2);
            expect(err.message).toEqual(TEST_STATUS_MESSAGE);
            done();
        });
    });
    it("unimplemented_method", function (done) {
        client.unimplementedCall(new Empty(), null, (err) => {
            expect(err).toBeDefined();
            expect('code' in err).toBeTrue();
            expect(err.code).toEqual(12);
            done();
        });
    })
    it("unimplemented_service", function (done) {
        const badClient = new UnimplementedServiceClient(SERVER_HOST, null, null);
        badClient.unimplementedCall(new Empty(), null, (err) => {
            expect(err).toBeDefined();
            expect('code' in err).toBeTrue();
            // TODO: enable this check after we decided the behaviour
            // expect(err.code).toEqual(12);
            done();
        });
    })
})

