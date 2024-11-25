package httpmock_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/amidgo/httpmock"
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

type ServerTest struct {
	CaseName         string
	TestReporterMock func(m *testReporterMock)
	Calls            []httpmock.Call
	Execute          func(server *httptest.Server)
}

func (s *ServerTest) Name() string {
	return s.CaseName
}

func (s *ServerTest) Test(t *testing.T) {
	rp := &testReporterMock{t: t}

	if s.TestReporterMock != nil {
		s.TestReporterMock(rp)
	}

	server := httpmock.NewServer(rp, s.Calls...)

	if s.Execute != nil {
		s.Execute(server)
	}
}

func Test_Server(t *testing.T) {
	header := make(http.Header)

	header.Set("X-My-Header", "Hello")
	header.Add("X-My-Headers", "Hello")
	header.Add("X-My-Headers", "Hello")
	header.Add("X-My-Headers", "Hello")

	tester.RunNamedTesters(t,
		&ServerTest{
			CaseName: "basic call with Hello World! body",
			TestReporterMock: func(m *testReporterMock) {
				m.ExpectSuccess()
			},
			Calls: []httpmock.Call{
				{
					Input: httpmock.Input{
						Body:   httpmock.RawBody(string("Hello World!")),
						URL:    mustParseURL("http://localhost:1000/any/target?key=value&key=value&name=Dima"),
						Header: header,
					},
				},
			},
			Execute: func(server *httptest.Server) {
				client := server.Client()

				req, err := http.NewRequest(http.MethodPost, server.URL+"/any/target?key=value&key=value&name=Dima", strings.NewReader("Hello World!"))
				if err != nil {
					return
				}

				req.Header = header

				client.Do(req)
			},
		},
		&ServerTest{
			CaseName: "expect zero calls but one times executed",
			TestReporterMock: func(m *testReporterMock) {
				calls := []testReporterCall{
					{
						format: "unexpected call to zero calls handler",
					},
				}

				m.ExpectErrorfCalls(calls)
			},
			Execute: func(server *httptest.Server) {
				client := server.Client()

				req, err := http.NewRequest(http.MethodGet, server.URL+"/any/target", http.NoBody)
				if err != nil {
					return
				}

				client.Do(req)
			},
		},
		&ServerTest{
			CaseName: "invalid body, header value, url query, url raw path",
			TestReporterMock: func(m *testReporterMock) {
				calls := []testReporterCall{
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
			Calls: []httpmock.Call{
				{
					Input: httpmock.Input{
						Body:   httpmock.RawBody(string("HelloWorld!")),
						URL:    mustParseURL("http://localhost:1000/any/targt?key=value"),
						Header: header,
					},
				},
			},
			Execute: func(server *httptest.Server) {
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
