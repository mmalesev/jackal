/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"testing"
	"time"

	"github.com/scionproto/scion/go/lib/sciond"
	"github.com/scionproto/scion/go/lib/sock/reliable"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestTransportConfig(t *testing.T) {
	rawCfg := `
bind_addr 0.0.0.0
`
	trCfg := TransportConfig{}
	err := yaml.Unmarshal([]byte(rawCfg), &trCfg)
	require.NotNil(t, err)

	rawCfg = `
bind_addr: 0.0.0.0
`
	err = yaml.Unmarshal([]byte(rawCfg), &trCfg)
	require.Nil(t, err)
	require.Equal(t, "0.0.0.0", trCfg.BindAddress)
	require.Equal(t, 5269, trCfg.Port)
	require.Equal(t, time.Duration(600)*time.Second, trCfg.KeepAlive)

	rawCfg = `
bind_addr: 127.0.0.1
port: 5999
keep_alive: 200
`
	err = yaml.Unmarshal([]byte(rawCfg), &trCfg)
	require.Nil(t, err)
	require.Equal(t, "127.0.0.1", trCfg.BindAddress)
	require.Equal(t, 5999, trCfg.Port)
	require.Equal(t, time.Duration(200)*time.Second, trCfg.KeepAlive)
}

func TestConfig(t *testing.T) {
	cfg := Config{}
	rawCfg := `
dial_timeout: 300
connect_timeout: 250
max_stanza_size: 8192
`
	err := yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.NotNil(t, err) // missing dialback secret

	rawCfg = `
dialback_secret: s3cr3t
`
	err = yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.Nil(t, err) // defaults
	require.Equal(t, defaultDialTimeout, cfg.DialTimeout)
	require.Equal(t, defaultConnectTimeout, cfg.ConnectTimeout)
	require.Equal(t, defaultMaxStanzaSize, cfg.MaxStanzaSize)

	rawCfg = `
dialback_secret: s3cr3t
dial_timeout: 300
connect_timeout: 250
max_stanza_size: 8192
`
	err = yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.Nil(t, err) // defaults
	require.Equal(t, time.Duration(300)*time.Second, cfg.DialTimeout)
	require.Equal(t, time.Duration(250)*time.Second, cfg.ConnectTimeout)
	require.Equal(t, 8192, cfg.MaxStanzaSize)
	require.Nil(t, cfg.Scion)
	rawCfg += `
scion_transport:
  addr: "my_scion_address"
  privkey_path: "key_path"
  cert_path: "certificate_path"
`
	err = yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.NotNil(t, cfg.Scion)
}

func TestScionConfig(t *testing.T) {
	cfg := ScionConfig{}
	rawCfg := `empty`
	err := yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.NotNil(t, err)

	rawCfg = `
addr: "scion_addr"
`
	err = yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.NotNil(t, err)
	require.Equal(t, cfg.Address, "scion_addr")

	rawCfg += `
privkey_path: "key_path"
cert_path: "c_path"
`
	err = yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.NotNil(t, cfg)
	require.Equal(t, cfg.Key, "key_path")
	require.Equal(t, cfg.Cert, "c_path")
	require.Equal(t, cfg.Port, defaultScionTransportPort)
	require.Equal(t, cfg.KeepAlive, defaultTransportKeepAlive)
	require.Equal(t, cfg.Dispatcher, reliable.DefaultDispPath)
	require.Equal(t, cfg.Sciond, sciond.DefaultSCIONDPath)

	rawCfg += `
port: 1234
dispatcher_path: "d_path"
sciond_path: "s_path"
keep_alive: 17
`
	err = yaml.Unmarshal([]byte(rawCfg), &cfg)
	require.NotNil(t, cfg)
	require.Equal(t, cfg.Port, 1234)
	require.Equal(t, cfg.KeepAlive, 17*time.Second)
	require.Equal(t, cfg.Dispatcher, "d_path")
	require.Equal(t, cfg.Sciond, "s_path")
}
