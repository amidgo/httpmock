package httpmock

import "fmt"

type TestReporter interface {
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)

	Cleanup(func())
}

func testReporterWithCallNumber(t TestReporter, number int64) TestReporter {
	return prefixTestReporter{
		t:      t,
		prefix: fmt.Sprintf("%d call, ", number),
	}
}

type prefixTestReporter struct {
	t      TestReporter
	prefix string
}

func (p prefixTestReporter) Errorf(format string, args ...any) {
	p.t.Errorf(p.prefix+format, args...)
}

func (p prefixTestReporter) Fatalf(format string, args ...any) {
	p.t.Fatalf(p.prefix+format, args...)
}

func (p prefixTestReporter) Cleanup(f func()) {
	p.t.Cleanup(f)
}
