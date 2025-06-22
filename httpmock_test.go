package httpmock

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"golang.org/x/sync/errgroup"
)

type transportTest struct {
	Name         string
	TestReporter func(t *testing.T) TestReporter
	Calls        Calls
	Execute      func(client *http.Client) error
}

func (s *transportTest) Test(t *testing.T) {
	tr := s.TestReporter(t)

	if s.Calls == nil {
		t.Fatalf("unexpected empty calls")

		return
	}

	client := &http.Client{
		Transport: NewTransport(tr, s.Calls, HandleCallCompareInput),
	}

	if s.Execute != nil {
		err := s.Execute(client)
		if err != nil {
			t.Fatalf("execute, receive unexpected error, %s", err)
		}
	}
}

func runTransportTests(t *testing.T, tests ...*transportTest) {
	for _, tst := range tests {
		t.Run(tst.Name, tst.Test)
	}
}

func Test_Transport(t *testing.T) {
	header := make(http.Header)

	header.Set("X-My-Header", "Hello")
	header.Add("X-My-Headers", "Hello")
	header.Add("X-My-Headers", "Hello")
	header.Add("X-My-Headers", "Hello")

	runTransportTests(t,
		&transportTest{
			Name:         "basic call with Hello World! body and simple response",
			TestReporter: ExpectSuccessTestReporter,
			Calls: SequenceCalls(
				[]Call{
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
							StatusCode: http.StatusInternalServerError,
							Body:       RawBody("Not Found1"),
							Header:     header,
						},
					},
					{
						Input: Input{
							Method: http.MethodGet,
							Body:   RawBody("Hello World!2"),
							URL:    mustParseURL("http://localhost:1000/any/target"),
							Header: header,
						},
						Response: Response{
							StatusCode: http.StatusNotFound,
							Body:       RawBody("Not Found2"),
							Header:     header,
						},
					},
					{
						Input: Input{
							Method: http.MethodGet,
						},
						Response: Response{
							StatusCode: http.StatusNotFound,
						},
					},
				}...,
			),
			Execute: doMany(
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPut,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!1"),
						header: header,
					},
					Response{
						StatusCode: http.StatusInternalServerError,
						Body:       RawBody("Not Found1"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodGet,
						target: "/any/target",
						body:   strings.NewReader("Hello World!2"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found2"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodGet,
						target: "/any/target?key=value&key=value&name=Dima",
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
					},
				),
			),
		},
		&transportTest{
			Name:         "basic call with Hello World! body and simple response, static calls",
			TestReporter: ExpectSuccessTestReporter,
			Calls: StaticCalls(
				[]Call{
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
							StatusCode: http.StatusInternalServerError,
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
				}...,
			),
			Execute: doMany(
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPut,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!1"),
						header: header,
					},
					Response{
						StatusCode: http.StatusInternalServerError,
						Body:       RawBody("Not Found1"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodGet,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!2"),
						header: header,
					},

					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found2"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPut,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!1"),
						header: header,
					},
					Response{
						StatusCode: http.StatusInternalServerError,
						Body:       RawBody("Not Found1"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodGet,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!2"),
						header: header,
					},

					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found2"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPut,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!1"),
						header: header,
					},
					Response{
						StatusCode: http.StatusInternalServerError,
						Body:       RawBody("Not Found1"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodGet,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!2"),
						header: header,
					},

					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found2"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPut,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!1"),
						header: header,
					},
					Response{
						StatusCode: http.StatusInternalServerError,
						Body:       RawBody("Not Found1"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodGet,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!2"),
						header: header,
					},

					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found2"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPut,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!1"),
						header: header,
					},
					Response{
						StatusCode: http.StatusInternalServerError,
						Body:       RawBody("Not Found1"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodGet,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!2"),
						header: header,
					},

					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found2"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPut,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!1"),
						header: header,
					},
					Response{
						StatusCode: http.StatusInternalServerError,
						Body:       RawBody("Not Found1"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodGet,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!2"),
						header: header,
					},

					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found2"),
						Header:     header,
					},
				),
			),
		},
		&transportTest{
			Name:         "parallel execution, static server imitation",
			TestReporter: ExpectSuccessTestReporter,
			Calls: SequenceCalls(
				[]Call{
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
				}...,
			),
			Execute: doManyParallel(
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
			),
		},
		&transportTest{
			Name:         "parallel execution, static calls",
			TestReporter: ExpectSuccessTestReporter,
			Calls: StaticCalls(
				Call{
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
			),
			Execute: doManyParallel(
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
				do(
					request{
						method: http.MethodPost,
						target: "/any/target?key=value&key=value&name=Dima",
						body:   strings.NewReader("Hello World!"),
						header: header,
					},
					Response{
						StatusCode: http.StatusNotFound,
						Body:       RawBody("Not Found"),
						Header:     header,
					},
				),
			),
		},
		&transportTest{
			Name: "expect zero calls but one times executed",
			TestReporter: ExpectFailureTestReporter(
				nil,
				[]testReporterCall{
					{
						format: "no expected calls left",
					},
				},
			),
			Calls: SequenceCalls(),
			Execute: doUncheckedResponse(
				request{
					method: http.MethodGet,
					target: "/any/target",
					body:   http.NoBody,
				},
			),
		},
		&transportTest{
			Name: "expect zero calls but one times executed, static calls",
			TestReporter: ExpectFailureTestReporter(
				nil,
				[]testReporterCall{
					{
						format: "no expected calls left",
					},
				},
			),
			Calls: StaticCalls(),
			Execute: doUncheckedResponse(
				request{
					method: http.MethodGet,
					target: "/any/target",
					body:   http.NoBody,
				},
			),
		},
		&transportTest{
			Name: "expect one call but no calls executed",
			TestReporter: ExpectFailureTestReporter(
				[]testReporterCall{
					{
						format: "assert handler calls, not all calls were handled",
					},
				},
				nil,
			),
			Calls:   SequenceCalls(Call{}),
			Execute: doMany(),
		},
		&transportTest{
			Name: "expect many calls but no calls executed",
			TestReporter: ExpectFailureTestReporter(
				[]testReporterCall{
					{
						format: "assert handler calls, not all calls were handled",
					},
				},
				nil,
			),
			Calls:   SequenceCalls(make([]Call, 100)...),
			Execute: doMany(),
		},
		&transportTest{
			Name: "invalid body, header value, url query, url raw path",
			TestReporter: ExpectFailureTestReporter(
				[]testReporterCall{
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
						format: "1 call, wrong url query values by key %s, expect [%s], actual [%s]",
						args: []any{
							"key",
							"value",
							"",
						},
					},
					{
						format: "1 call, body not equal, expected %s actual %s",
						args: []any{
							"HelloWorld!",
							"Hello World!",
						},
					},
					{
						format: "1 call, wrong header values by key %s, expect [%s], actual [%s]",
						args: []any{
							"X-My-Header",
							"Hello",
							"",
						},
					},
					{
						format: "1 call, wrong header values by key %s, expect [%s], actual [%s]",
						args: []any{
							"X-My-Headers",
							"Hello,Hello,Hello",
							"",
						},
					},
				},
				nil,
			),
			Calls: SequenceCalls(
				[]Call{
					{
						Input: Input{
							Method: http.MethodPut,
							Body:   RawBody("HelloWorld!"),
							URL:    mustParseURL("http://localhost:1000/any/targt?key=value"),
							Header: header,
						},
					},
				}...,
			),
			Execute: doUncheckedResponse(
				request{
					method: http.MethodPost,
					target: "/any/target",
					body:   strings.NewReader("Hello World!"),
				},
			),
		},
		&transportTest{
			Name:         "do error in response",
			TestReporter: ExpectSuccessTestReporter,
			Calls: StaticCalls(
				Call{
					Input: Input{
						Method: http.MethodGet,
						Body:   RawBody{},
						Header: header,
						URL:    mustParseURL("http://localhost:1000/getInfo"),
					},
					DoError: io.ErrUnexpectedEOF,
				},
			),
			Execute: doManyParallel(
				doExpectError(
					request{
						method: http.MethodGet,
						header: header,
						target: "/getInfo",
					},
					io.ErrUnexpectedEOF,
				),
				doExpectError(
					request{
						method: http.MethodGet,
						header: header,
						target: "/getInfo",
					},
					io.ErrUnexpectedEOF,
				),
				doExpectError(
					request{
						method: http.MethodGet,
						header: header,
						target: "/getInfo",
					},
					io.ErrUnexpectedEOF,
				),
			),
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

type request struct {
	method string
	target string
	body   io.Reader
	header http.Header
}

func doMany(funcs ...func(client *http.Client) error) func(client *http.Client) error {
	return func(client *http.Client) error {
		for _, f := range funcs {
			err := f(client)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func doManyParallel(funcs ...func(client *http.Client) error) func(client *http.Client) error {
	return func(client *http.Client) error {
		errgroup := errgroup.Group{}

		for _, f := range funcs {
			errgroup.Go(
				func() error {
					return f(client)
				},
			)
		}

		return errgroup.Wait()
	}
}

func do(req request, expectedResponse Response) func(client *http.Client) error {
	return func(client *http.Client) error {
		header := req.header

		req, err := http.NewRequest(req.method, req.target, req.body)
		if err != nil {
			return fmt.Errorf("make request request, %w", err)
		}

		req.Header = header

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("do request, %w", err)
		}

		defer resp.Body.Close()

		tr := &testReporterMock{}

		compareStatusCode(tr, resp.StatusCode, expectedResponse.StatusCode)
		compareBody(tr, resp.Body, expectedResponse.Body)
		compareHeader(tr, resp.Header, expectedResponse.Header)

		errs := make([]error, 0, len(tr.errorfCalls))

		for _, cl := range tr.errorfCalls {
			errs = append(errs, fmt.Errorf(cl.format, cl.args...))
		}

		return errors.Join(errs...)
	}
}

func doExpectError(req request, expectedError error) func(client *http.Client) error {
	return func(client *http.Client) error {
		header := req.header

		req, err := http.NewRequest(req.method, req.target, req.body)
		if err != nil {
			return fmt.Errorf("make request, unexpected error, %s", err)
		}

		req.Header = header

		_, err = client.Do(req)
		if !errors.Is(err, expectedError) {
			return fmt.Errorf("doExpectError failed, expect %w, actual %w", expectedError, err)
		}

		return nil
	}
}

func doUncheckedResponse(req request) func(client *http.Client) error {
	return func(client *http.Client) error {
		header := req.header

		req, err := http.NewRequest(req.method, req.target, req.body)
		if err != nil {
			return fmt.Errorf("make request, unexpected error, %s", err)
		}

		req.Header = header

		_, _ = client.Do(req)

		return nil
	}
}

type testReporterCall struct {
	format string
	args   []any
}

type testReporterMock struct {
	mu          sync.Mutex
	t           *testing.T
	errorfCalls []testReporterCall
	fatalfCalls []testReporterCall
}

func ExpectFailureTestReporter(errorfCalls, fatalfCalls []testReporterCall) func(t *testing.T) TestReporter {
	return func(t *testing.T) TestReporter {
		tr := &testReporterMock{
			t: t,
		}

		t.Cleanup(
			func() {
				tr.mu.Lock()
				defer tr.mu.Unlock()

				if !reflect.DeepEqual(tr.errorfCalls, errorfCalls) {
					tr.t.Errorf("errorf calls not equal,\nexpected %v,\n\nactual %v", errorfCalls, tr.errorfCalls)
				}

				if !reflect.DeepEqual(tr.fatalfCalls, fatalfCalls) {
					tr.t.Errorf("fatalf calls not equal,\nexpected %v,\n\nactual %v", fatalfCalls, tr.fatalfCalls)
				}
			},
		)

		return tr
	}
}

func ExpectSuccessTestReporter(t *testing.T) TestReporter {
	tr := &testReporterMock{
		t: t,
	}

	t.Cleanup(func() {
		tr.mu.Lock()
		defer tr.mu.Unlock()

		if tr.errorfCalls != nil {
			tr.t.Errorf("expect zero errorf calls, actual %v", tr.errorfCalls)
		}
	})

	return tr
}

func (tm *testReporterMock) Fatalf(format string, args ...any) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.fatalfCalls = append(tm.fatalfCalls, testReporterCall{format: format, args: args})
}

func (tm *testReporterMock) Errorf(format string, args ...any) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.errorfCalls = append(tm.errorfCalls, testReporterCall{format: format, args: args})
}

func (tm *testReporterMock) Cleanup(f func()) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.t.Cleanup(f)
}

func compareStatusCode(t TestReporter, actual, expected int) {
	if actual != expected {
		t.Errorf("wrong response status code, expected %d, actual %d", expected, actual)
	}
}

type bodyTest struct {
	Name          string
	Body          Body
	ExpectedBytes []byte
}

func (b *bodyTest) Test(t *testing.T) {
	bytes, err := b.Body.Bytes()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !slices.Equal(bytes, b.ExpectedBytes) {
		t.Errorf("compare b.Body.Bytes bytes, expected %v, actual %s", string(b.ExpectedBytes), string(bytes))
	}
}

func Test_Body(t *testing.T) {
	type jsonValue struct {
		Name string `json:"name"`
	}

	tests := []*bodyTest{
		{
			Name:          "raw body",
			Body:          RawBody("Hello World!"),
			ExpectedBytes: []byte("Hello World!"),
		},
		{
			Name:          "json body",
			Body:          JSONBody(jsonValue{Name: "amidman"}),
			ExpectedBytes: []byte(`{"name":"amidman"}`),
		},
	}

	for _, tst := range tests {
		t.Run(tst.Name, tst.Test)
	}
}
