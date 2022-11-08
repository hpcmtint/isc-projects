package testutil

import (
	"io"
	"os"
	"strings"
	"time"

	errors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Capture the stdout (including log output) and stderr content produced by
// a given function.
func CaptureOutput(f func()) (stdout []byte, stderr []byte, err error) {
	rescueStdout := os.Stdout
	rescueStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr
	rescueLogOutput := logrus.StandardLogger().Out
	logrus.StandardLogger().SetOutput(wOut)
	// Restore the standard pipelines at the end.
	defer func() {
		os.Stdout = rescueStdout
		os.Stderr = rescueStderr
		logrus.StandardLogger().SetOutput(rescueLogOutput)
	}()

	// Execute function
	f()

	// Close the internal pipelines.
	wOut.Close()
	wErr.Close()

	// Reads the stdout
	stdout, err = io.ReadAll(rOut)
	if err != nil {
		err = errors.Wrap(err, "cannot read stdout")
		return
	}

	stderr, err = io.ReadAll(rErr)
	err = errors.Wrap(err, "cannot read stderr")
	return stdout, stderr, err
}

// Parses the conventional filename with suffix.
// Returns a prefix of filename, parsed timestamp, and error (if failed).
func ParseTimestampFilename(filename string) (prefix string, timestamp time.Time, extension string, err error) {
	timestampStart := strings.LastIndex(filename, "_")
	if timestampStart <= 0 {
		err = errors.Errorf("missing prefix delimiter: %s", filename)
		return
	}
	timestampStart++
	prefix = filename[:timestampStart]

	timestampEnd := strings.Index(filename[timestampStart:], ".")
	if timestampEnd >= 0 {
		timestampEnd += timestampStart
		extension = filename[timestampEnd:]
	} else {
		timestampEnd = len(filename)
	}

	if timestampEnd-timestampStart < len(time.RFC3339)-5 { // Timezone is optional
		err = errors.Errorf("timestamp is too short: %s", filename)
		return
	}

	raw := filename[timestampStart:timestampEnd]
	raw = raw[:11] + strings.ReplaceAll(raw[11:], "-", ":")

	timestamp, err = time.Parse(time.RFC3339, raw)
	err = errors.Wrapf(err, "cannot parse a timestamp: %s for: %s", raw, filename)
	return
}

func CreateEnvironmentRestorePoint() func() {
	originalEnv := os.Environ()

	return func() {
		originalEnvDict := make(map[string]string, len(originalEnv))
		for _, pair := range originalEnv {
			key, value, _ := strings.Cut(pair, "=")
			originalEnvDict[key] = value
		}

		actualEnv := os.Environ()
		actualKeys := make(map[string]bool, len(actualEnv))
		for _, actualPair := range actualEnv {
			actualKey, actualValue, _ := strings.Cut(actualPair, "=")
			actualKeys[actualKey] = true
			originalValue, exist := originalEnvDict[actualKey]

			if !exist {
				// Environment variable was added.
				os.Unsetenv(actualKey)
			} else if actualValue != originalValue {
				// Environment variable was changed.
				os.Setenv(actualKey, originalValue)
			}
		}

		for originalKey, originalValue := range originalEnvDict {
			if _, exist := actualKeys[originalKey]; !exist {
				// Environment variable was removed.
				os.Setenv(originalKey, originalValue)
			}
		}
	}
}
