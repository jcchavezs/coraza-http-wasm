package main

import (
	"testing"

	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"
	"github.com/stretchr/testify/require"
)

type mockAPIHost struct {
	api.Host
	t         *testing.T
	getConfig func() []byte
}

func (h mockAPIHost) GetConfig() []byte {
	return h.getConfig()
}

func (h mockAPIHost) LogEnabled(api.LogLevel) bool {
	return h.t != nil
}

func (h mockAPIHost) Log(_ api.LogLevel, msg string) {
	h.t.Log(msg)
}

func TestGetDirectivesFromHost(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		cfg, err := getConfigFromHost(mockAPIHost{t: t, getConfig: func() []byte {
			return nil
		}})
		require.True(t, cfg.includeCRS)
		require.Empty(t, cfg.directives)
		require.NoError(t, err)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := getConfigFromHost(mockAPIHost{getConfig: func() []byte {
			return []byte("abcd")
		}})
		require.ErrorContains(t, err, "invalid host config")
	})

	t.Run("invalid directives value", func(t *testing.T) {
		_, err := getConfigFromHost(mockAPIHost{getConfig: func() []byte {
			return []byte("{\"directives\": true}")
		}})
		require.ErrorContains(t, err, "invalid host config")
	})

	t.Run("empty directives", func(t *testing.T) {
		_, err := getConfigFromHost(mockAPIHost{getConfig: func() []byte {
			return []byte("{\"directives\": []}")
		}})
		require.ErrorContains(t, err, "empty directives")
	})

	t.Run("valid directives", func(t *testing.T) {
		cfg, err := getConfigFromHost(mockAPIHost{getConfig: func() []byte {
			return []byte(`
			{
				"directives": [
					"SecRuleEngine: On",
					"SecDebugLog /etc/var/logs/coraza.conf"
				]
			}
			`)
		}})
		require.True(t, cfg.includeCRS)
		require.NoError(t, err)
		require.Equal(t, "SecRuleEngine: On\nSecDebugLog /etc/var/logs/coraza.conf", cfg.directives)
	})
}

func TestInitializeWAF(t *testing.T) {
	_, err := initializeWAF(mockAPIHost{t: t, getConfig: func() []byte {
		return []byte(`
		{
			"directives": [
				"SecRuleEngine On",
				"SecDebugLog /dev/stdout",
				"SecDebugLogLevel 9",
				"SecRule REQUEST_URI \"@rx .\" \"phase:1,deny,status:403,id:'1234'\""
			]
		}`)
	}})
	require.NoError(t, err)
}
