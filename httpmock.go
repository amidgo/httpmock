package httpmock

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"time"
)

type Body interface {
	Bytes() ([]byte, error)
}

type RawBody []byte

func (r RawBody) Bytes() ([]byte, error) {
	return r, nil
}

type jsonBody struct{ value any }

func (j jsonBody) Bytes() ([]byte, error) {
	return json.Marshal(j.value)
}

func JSONBody(value any) Body {
	return jsonBody{value: value}
}

type Call struct {
	Input    Input
	Response Response
	DoError  error
	Delay    time.Duration
}

type Input struct {
	Method string
	Body   Body
	Header http.Header
	URL    *url.URL
}

type Response struct {
	StatusCode int
	Body       Body
	Header     http.Header
}

type Calls interface {
	// minimum called times is 1
	Call(calledTimes int) (Call, bool)

	Done(calledTimes int) bool
}

type sequenceCalls []Call

func SequenceCalls(calls ...Call) Calls {
	return sequenceCalls(calls)
}

func (s sequenceCalls) Call(calledTimes int) (Call, bool) {
	calledTimes--

	if calledTimes >= len(s) {
		return Call{}, false
	}

	return s[calledTimes], true
}

func (s sequenceCalls) Done(calledTimes int) bool {
	if len(s) == 0 {
		return true
	}

	return calledTimes == len(s)
}

type staticCalls []Call

func StaticCalls(calls ...Call) Calls {
	return staticCalls(calls)
}

func (s staticCalls) Call(calledTimes int) (Call, bool) {
	if len(s) == 0 {
		return Call{}, false
	}

	if len(s) == 1 || calledTimes == 1 {
		return s[0], true
	}

	index := (calledTimes - 1 + len(s)) % len(s)

	return s[index], true
}

func (staticCalls) Done(int) bool {
	return true
}

type HandleCall func(t TestReporter, w http.ResponseWriter, r *http.Request, call Call)

type transport struct {
	t           TestReporter
	calledTimes atomic.Int64
	handleCall  func(t TestReporter, w http.ResponseWriter, r *http.Request, call Call)
	calls       Calls
}

func NewTransport(t TestReporter, calls Calls, handleCall HandleCall) http.RoundTripper {
	ts := &transport{
		t:          t,
		calls:      calls,
		handleCall: handleCall,
	}

	t.Cleanup(ts.assert)

	return ts
}

func (h *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	calledTimes := h.calledTimes.Add(1)

	t := testReporterWithCallNumber(h.t, calledTimes)

	call, ok := h.calls.Call(int(calledTimes))
	if !ok {
		h.t.Fatalf("no expected calls left")

		return &http.Response{}, nil
	}

	if call.DoError != nil {
		return nil, call.DoError
	}

	w := httptest.NewRecorder()

	handleCall := HandleCallCompareInput
	if h.handleCall != nil {
		handleCall = h.handleCall
	}

	handleCall(t, w, r, call)

	return w.Result(), nil
}

func (h *transport) assert() {
	calledTimes := h.calledTimes.Load()

	if !h.calls.Done(int(calledTimes)) {
		h.t.Errorf("assert handler calls, not all calls were handled")
	}
}

func HandleCallCompareInput(t TestReporter, w http.ResponseWriter, r *http.Request, call Call) {
	compareInput(t, r, call.Input)
	writeResponse(t, w, call.Response)

	if call.Delay > 0 {
		<-time.After(call.Delay)
	}
}

func compareInput(t TestReporter, r *http.Request, input Input) {
	compareMethod(t, r.Method, input.Method)
	compareURL(t, r.URL, input.URL)
	compareBody(t, r.Body, input.Body)
	compareHeader(t, r.Header, input.Header)
}

func compareMethod(t TestReporter, requestMethod, inputMethod string) {
	if requestMethod != inputMethod {
		t.Errorf("wrong r.Method, expected %s, actual %s", inputMethod, requestMethod)
	}
}

func compareURL(t TestReporter, requestURL, inputURL *url.URL) {
	if inputURL == nil {
		return
	}

	if requestURL.Path != inputURL.Path {
		t.Errorf("wrong url.Path, expected %s, actual %s", inputURL.Path, requestURL.Path)
	}

	compareQuery(t, requestURL.Query(), inputURL.Query())
}

func compareQuery(t TestReporter, requestQuery, inputQuery url.Values) {
	if len(inputQuery) == 0 {
		return
	}

	keys := make([]string, 0, len(inputQuery))

	for key := range inputQuery {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	for _, key := range keys {
		inputQueryKeyValues := inputQuery[key]
		requestQueryKeyValues := requestQuery[key]

		if !slices.Equal(requestQueryKeyValues, inputQueryKeyValues) {
			t.Errorf(
				"wrong url query values by key %s, expect [%s], actual [%s]",
				key,
				strings.Join(inputQueryKeyValues, ","),
				strings.Join(requestQueryKeyValues, ","),
			)
		}
	}
}

func compareBody(t TestReporter, requestBody io.Reader, inputBody Body) {
	if requestBody == nil {
		requestBody = io.NopCloser(new(bytes.Reader))
	}

	bodyBytes, err := io.ReadAll(requestBody)
	if err != nil {
		t.Errorf("read body from request, %s", err)

		return
	}

	if inputBody == nil {
		inputBody = RawBody{}
	}

	inputBodyBytes, err := inputBody.Bytes()
	if err != nil {
		t.Errorf("read input body, %s", err)

		return
	}

	if !slices.Equal(inputBodyBytes, bodyBytes) {
		t.Errorf("body not equal, expected %s actual %s", string(inputBodyBytes), string(bodyBytes))
	}
}

func compareHeader(t TestReporter, requestHeader, inputHeader http.Header) {
	keys := make([]string, 0, len(inputHeader))
	for key := range inputHeader {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	for _, key := range keys {
		requestHeaderKeyValues := requestHeader.Values(key)
		values := inputHeader.Values(key)

		if !slices.Equal(requestHeaderKeyValues, values) {
			t.Errorf("wrong header values by key %s, expect [%s], actual [%s]",
				key,
				strings.Join(values, ","),
				strings.Join(requestHeaderKeyValues, ","),
			)
		}
	}
}

func writeResponse(t TestReporter, w http.ResponseWriter, response Response) {
	writeResponseHeader(w, response)
	writeResponseBody(t, w, response.Body)
}

func writeResponseHeader(w http.ResponseWriter, response Response) {
	header := response.Header

	if response.Header == nil {
		header = make(http.Header)
	}

	for key := range header {
		for _, value := range header.Values(key) {
			w.Header().Add(key, value)
		}
	}

	if response.StatusCode == 0 {
		response.StatusCode = http.StatusOK
	}

	w.WriteHeader(response.StatusCode)
}

func writeResponseBody(t TestReporter, w http.ResponseWriter, body Body) {
	if body == nil {
		body = RawBody{}
	}

	bytes, err := body.Bytes()
	if err != nil {
		t.Errorf("read body bytes, unexpected error: %s", err)

		return
	}

	_, err = w.Write(bytes)
	if err != nil {
		t.Errorf("write body bytes to response writer, unexpected error: %s", err)
	}
}
