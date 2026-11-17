package database

import (
	"bytes"
	stdlog "log"
	"os"
	"strings"
	"testing"

	"github.com/exgamer/gosdk-core/pkg/logger"
)

// captureLog Лог gosdk-core в буфер на уровне info на время теста
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()

	prevLevel := logger.GetLevel()
	buf := &bytes.Buffer{}

	logger.SetLogger(stdlog.New(buf, "", 0))
	logger.SetLevel(logger.LevelInfo)

	t.Cleanup(func() {
		logger.SetLogger(stdlog.New(os.Stdout, "", 0))
		logger.SetLevel(prevLevel)
	})

	return buf
}

func TestCheckSslMode(t *testing.T) {
	cases := []struct {
		name     string
		sslMode  string
		appEnv   string
		wantWarn bool
	}{
		{name: "пусто на prod — WARN", appEnv: "prod", wantWarn: true},
		{name: "disable на prod — WARN", sslMode: "disable", appEnv: "prod", wantWarn: true},
		{name: "allow на stage — WARN", sslMode: "allow", appEnv: "stage", wantWarn: true},
		{name: "prefer — WARN", sslMode: "Prefer", appEnv: "prod", wantWarn: true},
		{name: "пустой APP_ENV — не local", sslMode: "disable", wantWarn: true},
		{name: "неизвестный APP_ENV — не local", sslMode: "disable", appEnv: "locale", wantWarn: true},
		{name: "local — без WARN", sslMode: "disable", appEnv: "local"},
		{name: "require на prod — без WARN", sslMode: "require", appEnv: "prod"},
		{name: "verify-full на prod — без WARN", sslMode: "verify-full", appEnv: "prod"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetSslModeWarned()
			buf := captureLog(t)

			checkSslMode(tc.sslMode, tc.appEnv)

			warns := strings.Count(buf.String(), "may be unencrypted")
			if want := map[bool]int{true: 1, false: 0}[tc.wantWarn]; warns != want {
				t.Errorf("warnings = %d, want %d; log %q", warns, want, buf.String())
			}
		})
	}
}

// resetSslModeWarned Забыть выданные WARN: каждый случай проверяет свой
func resetSslModeWarned() {
	sslModeWarned.Range(func(k, _ any) bool {
		sslModeWarned.Delete(k)

		return true
	})
}

func TestCheckSslMode_WarnsOnce(t *testing.T) {
	resetSslModeWarned()
	t.Cleanup(resetSslModeWarned)

	buf := captureLog(t)

	for i := 0; i < 3; i++ {
		checkSslMode("disable", "prod")
	}

	if n := strings.Count(buf.String(), "may be unencrypted"); n != 1 {
		t.Fatalf("warnings = %d, want 1; log %q", n, buf.String())
	}
}
