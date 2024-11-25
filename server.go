package httpmock

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync/atomic"
	"time"
)

type Call struct {
	Input    Input
	Response Response
	Delay    time.Duration
}

type Input struct {
	Body   Body
	Header http.Header
	URL    *url.URL
}

type Response struct {
	StatusCode int
	Body       Body
	Header     http.Header
}

func JSONContentTypeHeader(header http.Header) http.Header {
	return ContentTypeHeader("application/json", header)
}

func ContentTypeHeader(contentType string, header http.Header) http.Header {
	header.Add("Content-Type", contentType)

	return header
}

func NewServer(t TestReporter, calls ...Call) *httptest.Server {
	if len(calls) == 0 {
		return httptest.NewServer(&zeroCallsHandler{t: t})
	}

	h := &handler{t: t, calls: calls}

	t.Cleanup(h.assert)

	return httptest.NewServer(h)
}

func NewStaticServer(t TestReporter, calls ...Call) *httptest.Server {
	if len(calls) == 0 {
		return httptest.NewServer(&zeroCallsHandler{t: t})
	}

	return httptest.NewServer(
		&staticHandler{
			t:     t,
			calls: calls,
		},
	)
}

type zeroCallsHandler struct {
	t TestReporter
}

func (z *zeroCallsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	z.t.Errorf("unexpected call to zero calls handler")
}

type handler struct {
	t           TestReporter
	calledTimes atomic.Int64
	calls       []Call
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	calledTimes := h.calledTimes.Add(1)

	t := testReporterWithCallNumber(h.t, calledTimes)

	if calledTimes > int64(len(h.calls)) {
		h.t.Fatalf("call count limit exceeded")

		return
	}

	call := h.calls[calledTimes-1]

	handleCall(t, w, r, call)
}

func (h *handler) assert() {
	calledTimes := h.calledTimes.Load()

	if calledTimes != int64(len(h.calls)) {
		h.t.Errorf("assert handler calls, not all calls were handled")
	}
}

type staticHandler struct {
	t           TestReporter
	calledTimes atomic.Int64
	calls       []Call
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	calledTimes := h.calledTimes.Add(1)

	t := testReporterWithCallNumber(h.t, calledTimes)

	call := h.nextCall(calledTimes)

	handleCall(t, w, r, call)
}

func (h *staticHandler) nextCall(calledTimes int64) Call {
	if len(h.calls) == 1 || calledTimes == 1 {
		return h.calls[0]
	}

	index := calledTimes % int64(len(h.calls))

	return h.calls[index]
}

func handleCall(t TestReporter, w http.ResponseWriter, r *http.Request, call Call) {
	compareInput(t, r, call.Input)
	writeResponse(t, w, call.Response)

	if call.Delay > 0 {
		<-time.After(call.Delay)
	}
}

func compareInput(t TestReporter, r *http.Request, input Input) {
	compareURL(t, r.URL, input.URL)
	compareBody(t, r.Body, input.Body)
	compareHeader(t, r.Header, input.Header)
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

	for key, values := range inputQuery {
		requestQueryKeyValues := requestQuery[key]

		if !slices.Equal(requestQueryKeyValues, values) {
			t.Errorf("wrong url query values by key %s, expect %v, actual %v", key, values, requestQueryKeyValues)
		}
	}
}

func compareBody(t TestReporter, requestBody io.Reader, inputBody Body) {
	bodyBytes, err := io.ReadAll(requestBody)
	if err != nil {
		t.Errorf("read body from request, %s", err)

		return
	}

	if inputBody == nil {
		inputBody = NoBody{}
	}

	inputBodyBytes := inputBody.Bytes()

	if !slices.Equal(inputBodyBytes, bodyBytes) {
		t.Errorf("body not equal,\nexpected %s\nactual %s", string(inputBodyBytes), string(bodyBytes))
	}
}

func compareHeader(t TestReporter, requestHeader, inputHeader http.Header) {
	for key := range inputHeader {
		requestHeaderKeyValues := requestHeader.Values(key)
		values := inputHeader.Values(key)

		if !slices.Equal(requestHeaderKeyValues, values) {
			t.Errorf("wrong header values by key %s, expect %v, actual %v", key, values, requestHeaderKeyValues)
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
			header.Add(key, value)
		}
	}

	if response.StatusCode == 0 {
		response.StatusCode = http.StatusOK
	}

	w.WriteHeader(response.StatusCode)
}

func writeResponseBody(t TestReporter, w http.ResponseWriter, body Body) {
	if body == nil {
		return
	}

	_, err := w.Write(body.Bytes())
	if err != nil {
		t.Errorf("write response bytes, %s", err)
	}
}
