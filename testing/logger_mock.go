package testing

import "fmt"

type MockLogger struct {
	Messages   []string
	FatalCalls int
	ErrorCalls int
	WarnCalls  int
	InfoCalls  int
	DebugCalls int
}

func (m *MockLogger) LogFatal(err error) {
	m.Messages = append(m.Messages, fmt.Sprintf("FATAL %s", err.Error()))
	m.FatalCalls++
}

func (m *MockLogger) LogError(err error) {
	m.Messages = append(m.Messages, fmt.Sprintf("ERROR %s", err.Error()))
	m.ErrorCalls++
}

func (m *MockLogger) LogWarn(message string) {
	m.Messages = append(m.Messages, fmt.Sprintf("WARNING %s", message))
	m.WarnCalls++
}

func (m *MockLogger) LogInfo(message string) {
	m.Messages = append(m.Messages, fmt.Sprintf("INFO %s", message))
	m.InfoCalls++
}

func (m *MockLogger) LogDebug(message string) {
	m.Messages = append(m.Messages, fmt.Sprintf("DEBUG %s", message))
	m.DebugCalls++
}
