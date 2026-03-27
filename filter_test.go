package zapp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseKeyValueFilter(t *testing.T) {
	t.Run("string_value", func(t *testing.T) {
		filter, err := ParseKeyValueFilter("logger=acme")
		require.NoError(t, err)
		require.Equal(t, "logger", filter.Key)
		require.Equal(t, "acme", filter.Value)
	})

	t.Run("json_number_value", func(t *testing.T) {
		filter, err := ParseKeyValueFilter("block_num=308267722")
		require.NoError(t, err)
		require.Equal(t, "block_num", filter.Key)
		require.Equal(t, float64(308267722), filter.Value)
	})

	t.Run("invalid_expression", func(t *testing.T) {
		_, err := ParseKeyValueFilter("logger")
		require.EqualError(t, err, `invalid filter "logger": expected key=value`)
	})
}

func TestProcessorKeyValueFilter(t *testing.T) {
	runLogTests(t, []logTest{
		{
			name: "matches_string_field",
			lines: []string{
				`{"severity":"INFO","timestamp":"2022-04-21T14:50:18.382974069-04:00","logger":"acme","message":"m"}`,
				`{"severity":"INFO","timestamp":"2022-04-21T14:50:18.382974069-04:00","logger":"other","message":"m"}`,
			},
			expectedLines: []string{
				"[2022-04-21 14:50:18.382 EDT] \x1b[32mINFO\x1b[0m \x1b[38;5;244m(acme)\x1b[0m \x1b[34mm\x1b[0m",
			},
			options: []ProcessorOption{WithKeyValueFilter(mustParseKeyValueFilter(t, "logger=acme"))},
		},
		{
			name: "matches_numeric_field",
			lines: []string{
				`{"level":"info","ts":1545445711.144533,"caller":"c","msg":"m","block":42}`,
				`{"level":"info","ts":1545445711.144533,"caller":"c","msg":"m","block":43}`,
			},
			expectedLines: []string{
				"[2018-12-21 21:28:31.144 EST] \x1b[32mINFO\x1b[0m \x1b[38;5;244m(c)\x1b[0m \x1b[34mm\x1b[0m {\"block\":42}",
			},
			options: []ProcessorOption{WithKeyValueFilter(mustParseKeyValueFilter(t, "block=42"))},
		},
		{
			name: "matches_any_duplicate_key_value",
			lines: []string{
				`{"level":"info","module":"server","module":"txindex","height":24855179,"time":"2025-07-17T14:18:17Z","message":"indexed block events"}`,
				`{"level":"info","module":"server","module":"database","height":24855179,"time":"2025-07-17T14:18:17Z","message":"indexed block events"}`,
			},
			expectedLines: []string{
				"[2025-07-17 10:18:17.000 EDT] \x1b[32mINFO\x1b[0m \x1b[38;5;244m(server.txindex)\x1b[0m \x1b[34mindexed block events\x1b[0m {\"height\":24855179}",
			},
			options: []ProcessorOption{WithKeyValueFilter(mustParseKeyValueFilter(t, "module=txindex"))},
		},
	})
}

func mustParseKeyValueFilter(t *testing.T, input string) *KeyValueFilter {
	t.Helper()

	filter, err := ParseKeyValueFilter(input)
	require.NoError(t, err)

	return filter
}
