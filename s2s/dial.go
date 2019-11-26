/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package s2s

import (
	"crypto/tls"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/transport"

	"github.com/lucas-clemente/quic-go"
	"github.com/netsec-ethz/scion-apps/lib/scionutil"
	"github.com/scionproto/scion/go/lib/sciond"
	"github.com/scionproto/scion/go/lib/snet"
	"github.com/scionproto/scion/go/lib/snet/squic"
	"github.com/scionproto/scion/go/lib/sock/reliable"

	libaddr "github.com/scionproto/scion/go/lib/addr"
)

type dialer struct {
	cfg         *Config
	router      *router.Router
	srvResolve  func(service, proto, name string) (cname string, addrs []*net.SRV, err error)
	dialTimeout func(network, address string, timeout time.Duration) (net.Conn, error)
}

func newDialer(cfg *Config, router *router.Router) *dialer {
	return &dialer{cfg: cfg, router: router, srvResolve: net.LookupSRV, dialTimeout: net.DialTimeout}
}

func (d *dialer) dialQUIC(remote *snet.Addr, localDomain, remoteDomain string) (*streamConfig, error) {
	var local *snet.Addr
	var err error
	if d.cfg.Scion.Address == "localhost" {
		local, err = scionutil.GetLocalhost()
	} else {
		local, err = snet.AddrFromString(d.cfg.Scion.Address)
	}
	if err != nil {
		return nil, err
	}

	sciondPath := sciond.GetDefaultSCIONDPath(nil)
	dispatcherPath := d.cfg.Scion.Dispatcher
	snet.Init(local.IA, sciondPath, reliable.NewDispatcherService(dispatcherPath))
	quicConfig := &quic.Config{
		KeepAlive: true,
	}
	sess, err := squic.DialSCION(nil, local, remote, quicConfig)
	if err != nil {
		return nil, err
	}
	biStream, err := sess.OpenStreamSync()
	if err != nil {
		log.Infof("Couldn't open a new QUIC Stream")
	}

	tr := transport.NewQUICSocketTransport(sess, biStream,
		d.cfg.Transport.KeepAlive)
	return &streamConfig{
		keyGen:        &keyGen{secret: d.cfg.DialbackSecret},
		localDomain:   localDomain,
		remoteDomain:  remoteDomain,
		transport:     tr,
		maxStanzaSize: d.cfg.MaxStanzaSize,
	}, nil
}

func (d *dialer) dialTCP(localDomain, remoteDomain string) (*streamConfig, error) {
	_, addrs, err := d.srvResolve("xmpp-server", "tcp", remoteDomain)
	if err != nil {
		log.Warnf("srv lookup error: %v", err)
	}
	var target string

	if err != nil || len(addrs) == 1 && addrs[0].Target == "." {
		target = remoteDomain + ":5269"
	} else {
		target = strings.TrimSuffix(addrs[0].Target, ".") + ":" + strconv.Itoa(int(addrs[0].Port))
	}
	conn, err := d.dialTimeout("tcp", target, d.cfg.DialTimeout)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		ServerName:   remoteDomain,
		Certificates: d.router.Certificates(),
	}
	tr := transport.NewSocketTransport(conn, d.cfg.Transport.KeepAlive)
	return &streamConfig{
		keyGen:        &keyGen{secret: d.cfg.DialbackSecret},
		localDomain:   localDomain,
		remoteDomain:  remoteDomain,
		transport:     tr,
		tls:           tlsConfig,
		maxStanzaSize: d.cfg.MaxStanzaSize,
	}, nil
}

func (d *dialer) dial(localDomain, remoteDomain string) (*streamConfig, error) {
	var ret *streamConfig
	var err error
	isSCIONAddress, remote := rainsLookup(remoteDomain)
	if isSCIONAddress {
		ret, err = d.dialQUIC(remote, localDomain, remoteDomain)
	} else {
		ret, err = d.dialTCP(localDomain, remoteDomain)
	}
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func rainsLookup(remoteDomain string) (bool, *snet.Addr) {
	host, port, err := net.SplitHostPort(remoteDomain)
	if err != nil {
		host = remoteDomain
		port = "52690"
	}
	ia, l3, err := scionutil.GetHostByName(host)
	if err != nil {
		return false, nil
	}

	p, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		p = 52690
	}
	l4 := libaddr.NewL4UDPInfo(uint16(p))
	raddr := &snet.Addr{IA: ia, Host: &libaddr.AppAddr{L3: l3, L4: l4}}

	return true, raddr
}
