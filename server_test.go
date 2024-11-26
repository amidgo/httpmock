package httpmock

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/amidgo/tester"
)

type testReporterCall struct {
	format string
	args   []any
}

type testReporterMock struct {
	t           *testing.T
	errorfCalls []testReporterCall
	fatalfCalls []testReporterCall
}

func (tm *testReporterMock) ExpectErrorfCalls(calls []testReporterCall) {
	tm.t.Cleanup(
		func() {
			if !reflect.DeepEqual(tm.errorfCalls, calls) {
				tm.t.Errorf("errorf calls not equal,\nexpected %v,\n\nactual   %v", calls, tm.errorfCalls)
			}
		},
	)
}

func (tm *testReporterMock) ExpectFatalfCalls(calls []testReporterCall) {
	tm.t.Cleanup(
		func() {
			if !reflect.DeepEqual(tm.fatalfCalls, calls) {
				tm.t.Errorf("fatalf calls not equal,\nexpected %v,\n\nactual %v", calls, tm.fatalfCalls)
			}
		},
	)
}

func (tm *testReporterMock) ExpectSuccess() {
	tm.t.Cleanup(
		func() {
			if tm.errorfCalls != nil {
				tm.t.Errorf("expect zero errorf calls, actual %v", tm.errorfCalls)
			}

			if tm.fatalfCalls != nil {
				tm.t.Errorf("expect zero fatalf calls, actual %v", tm.fatalfCalls)
			}
		},
	)
}

func (tm *testReporterMock) Fatalf(format string, args ...any) {
	tm.fatalfCalls = append(tm.fatalfCalls, testReporterCall{format: format, args: args})
}

func (tm *testReporterMock) Errorf(format string, args ...any) {
	tm.errorfCalls = append(tm.errorfCalls, testReporterCall{format: format, args: args})
}

func (tm *testReporterMock) Cleanup(f func()) {
	tm.t.Cleanup(f)
}

type serverTest struct {
	CaseName         string
	TestReporterMock func(m *testReporterMock)
	Calls            []Call
	Execute          func(t *testing.T, server *httptest.Server)
}

func (s *serverTest) Name() string {
	return s.CaseName
}

func (s *serverTest) Test(t *testing.T) {
	rp := &testReporterMock{t: t}

	if s.TestReporterMock != nil {
		s.TestReporterMock(rp)
	}

	server := NewServer(rp, s.Calls...)

	if s.Execute != nil {
		s.Execute(t, server)
	}
}

func Test_Server(t *testing.T) {
	header := make(http.Header)

	header.Set("X-My-Header", "Hello")
	header.Add("X-My-Headers", "Hello")
	header.Add("X-My-Headers", "Hello")
	header.Add("X-My-Headers", "Hello")

	tester.RunNamedTesters(t,
		&serverTest{
			CaseName: "basic call with Hello World! body and simple response",
			TestReporterMock: func(m *testReporterMock) {
				m.ExpectSuccess()
			},
			Calls: []Call{
				{
					Input: Input{
						Method: http.MethodPost,
						Body:   RawBody("Hello World!"),
						URL:    mustParseURL("http://localhost:1000/any/target?key=value&key=value&name=Dima"),
						Header: header,
					},
					Response: Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				},
				{
					Input: Input{
						Method: http.MethodPut,
						Body:   RawBody("Hello World!1"),
						URL:    mustParseURL("http://localhost:1000/any/target?key=value&key=value&name=Dima"),
						Header: header,
					},
					Response: Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found1"),
						Header:     header,
					},
				},
				{
					Input: Input{
						Method: http.MethodGet,
						Body:   RawBody("Hello World!2"),
						URL:    mustParseURL("http://localhost:1000/any/target?key=value&key=value&name=Dima"),
						Header: header,
					},
					Response: Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found2"),
						Header:     header,
					},
				},
			},
			Execute: func(t *testing.T, server *httptest.Server) {
				client := server.Client()

				req, err := http.NewRequest(http.MethodPost, server.URL+"/any/target?key=value&key=value&name=Dima", strings.NewReader("Hello World!"))
				if err != nil {
					return
				}

				req.Header = header

				do(t, client, req,
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				)

				req, err = http.NewRequest(http.MethodPut, server.URL+"/any/target?key=value&key=value&name=Dima", strings.NewReader("Hello World!1"))
				if err != nil {
					return
				}

				req.Header = header

				do(t, client, req,
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found1"),
						Header:     header,
					},
				)

				req, err = http.NewRequest(http.MethodGet, server.URL+"/any/target?key=value&key=value&name=Dima", strings.NewReader("Hello World!2"))
				if err != nil {
					return
				}

				req.Header = header

				do(t, client, req,
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found2"),
						Header:     header,
					},
				)
			},
		},
		&serverTest{
			CaseName: "expect zero calls but one times executed",
			TestReporterMock: func(m *testReporterMock) {
				calls := []testReporterCall{
					{
						format: "unexpected call to zero calls handler",
					},
				}

				m.ExpectErrorfCalls(calls)
			},
			Execute: func(_ *testing.T, server *httptest.Server) {
				client := server.Client()

				req, err := http.NewRequest(http.MethodGet, server.URL+"/any/target", http.NoBody)
				if err != nil {
					return
				}

				_, _ = client.Do(req)
			},
		},
		&serverTest{
			CaseName: "invalid body, header value, url query, url raw path",
			TestReporterMock: func(m *testReporterMock) {
				calls := []testReporterCall{
					{
						format: "1 call, wrong r.Method, expected %s, actual %s",
						args: []any{
							http.MethodPut,
							http.MethodPost,
						},
					},
					{
						format: "1 call, wrong url.Path, expected %s, actual %s",
						args: []any{
							"/any/targt",
							"/any/target",
						},
					},
					{
						format: "1 call, wrong url query values by key %s, expect %v, actual %v",
						args: []any{
							"key",
							[]string{"value"},
							[]string(nil),
						},
					},
					{
						format: "1 call, body not equal,\nexpected %s\nactual %s",
						args: []any{
							"HelloWorld!",
							"Hello World!",
						},
					},
					{
						format: "1 call, wrong header values by key %s, expect %v, actual %v",
						args: []any{
							"X-My-Header",
							[]string{"Hello"},
							[]string(nil),
						},
					},
					{
						format: "1 call, wrong header values by key %s, expect %v, actual %v",
						args: []any{
							"X-My-Headers",
							[]string{"Hello", "Hello", "Hello"},
							[]string(nil),
						},
					},
				}

				m.ExpectErrorfCalls(calls)
			},
			Calls: []Call{
				{
					Input: Input{
						Method: http.MethodPut,
						Body:   RawBody("HelloWorld!"),
						URL:    mustParseURL("http://localhost:1000/any/targt?key=value"),
						Header: header,
					},
				},
			},
			Execute: func(_ *testing.T, server *httptest.Server) {
				client := server.Client()

				req, err := http.NewRequest(http.MethodPost, server.URL+"/any/target", strings.NewReader("Hello World!"))
				if err != nil {
					return
				}

				_, _ = client.Do(req)
			},
		},
	)
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}

	return u
}

func do(t TestReporter, client *http.Client, req *http.Request, response Response) {
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("do request, unexpected error, %s", err)

		return
	}

	defer resp.Body.Close()

	statusCode := resp.StatusCode

	if statusCode != http.StatusNotFound {
		t.Errorf("wrong response status code, expected %d, actual %d", http.StatusNotFound, statusCode)
	}

	compareBody(t, resp.Body, response.Body)
	compareHeader(t, resp.Header, response.Header)
}
