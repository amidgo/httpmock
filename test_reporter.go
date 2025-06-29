package httpmock

import "fmt"

type TestReporter interface {
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)

	Cleanup(func())
}

func errorfTestReporterWithCallNumber(t TestReporter, number int64) TestReporter {
	return errorfPrefixTestReporter{
		TestReporter: t,
		prefix:       fmt.Sprintf("%d call, ", number),
	}
}

type errorfPrefixTestReporter struct {
	TestReporter
	prefix string
}

func (p errorfPrefixTestReporter) Errorf(format string, args ...any) {
	p.TestReporter.Errorf(p.prefix+format, args...)
}

type nilTestReporter struct{}

func (nilTestReporter) Fatalf(string, ...any) {}
func (nilTestReporter) Errorf(string, ...any) {}
func (nilTestReporter) Cleanup(func())        {}
