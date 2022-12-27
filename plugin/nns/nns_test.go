package nns

import (
	"context"
	"fmt"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient"
	"github.com/stretchr/testify/require"
)

func TestNNS(t *testing.T) {
	ctx := context.Background()
	//container := createDockerContainer(ctx, t, testImage)
	//defer container.Terminate(ctx)

	cli, err := rpcclient.New(ctx, "http://seed1t5.neo.org:20332", rpcclient.Options{})
	//require.NoError(t, err)
	//err  = cli.Init()
	////require.NoError(t, err)
	//cs, err := cli.GetContractStateByID(1)
	//require.NoError(t, err)
	contractHash, err := util.Uint160DecodeStringLE("50ac1c37690cc2cfc594472833cf57505d5f46de")
	if err != nil {
		fmt.Println(err)
	}

	nns := NNS{
		Next:         test.NextHandler(dns.RcodeSuccess, nil),
		Client:       cli,
		ContractHash: contractHash,
	}
	tests := []struct {
		qname         string
		qtype         uint16
		remote        string
		expectedCode  int
		expectedReply []string // ownernames for the records in the additional section.
		expectedErr   error
	}{
		{
			qname:         "wangmt.neo.ongoing.club",
			qtype:         dns.TypeA,
			expectedCode:  dns.RcodeSuccess,
			expectedReply: []string{"example.org.", "_udp.example.org."},
			expectedErr:   nil,
		},
		// Case insensitive and case preserving
		{
			qname:         "Example.ORG",
			qtype:         dns.TypeA,
			expectedCode:  dns.RcodeSuccess,
			expectedReply: []string{"Example.ORG.", "_udp.Example.ORG."},
			expectedErr:   nil,
		},
		{
			qname:         "example.org",
			qtype:         dns.TypeA,
			remote:        "2003::1/64",
			expectedCode:  dns.RcodeSuccess,
			expectedReply: []string{"example.org.", "_udp.example.org."},
			expectedErr:   nil,
		},
		{
			qname:         "Example.ORG",
			qtype:         dns.TypeA,
			remote:        "2003::1/64",
			expectedCode:  dns.RcodeSuccess,
			expectedReply: []string{"Example.ORG.", "_udp.Example.ORG."},
			expectedErr:   nil,
		},
	}

	for i, tc := range tests {
		req := new(dns.Msg)
		req.SetQuestion(dns.Fqdn(tc.qname), tc.qtype)
		rec := dnstest.NewRecorder(&test.ResponseWriter{RemoteIP: tc.remote})
		code, err := nns.ServeDNS(ctx, rec, req)
		if err != tc.expectedErr {
			t.Errorf("Test %d: Expected error %v, but got %v", i, tc.expectedErr, err)
		}
		if code != int(tc.expectedCode) {
			t.Errorf("Test %d: Expected status code %d, but got %d", i, tc.expectedCode, code)
		}
		if len(tc.expectedReply) != 0 {
			for i, expected := range tc.expectedReply {
				actual := rec.Msg.Extra[i].Header().Name
				if actual != expected {
					t.Errorf("Test %d: Expected answer %s, but got %s", i, expected, actual)
				}
			}
		}
	}
}

func TestGetNetmapHash(t *testing.T) {
	ctx := context.Background()
	container := createDockerContainer(ctx, t, testImage)
	defer container.Terminate(ctx)

	cli, err := rpcclient.New(ctx, "http://seed1t5.neo.org:20332", rpcclient.Options{})
	require.NoError(t, err)
	err = cli.Init()
	require.NoError(t, err)
	cs, err := cli.GetContractStateByID(1)
	require.NoError(t, err)

	nns := NNS{
		Next:         test.NextHandler(dns.RcodeSuccess, nil),
		Client:       cli,
		ContractHash: cs.Hash,
	}

	req := new(dns.Msg)
	req.SetQuestion(dns.Fqdn("netmap.neofs"), dns.TypeTXT) ///
	req.Question[0].Qclass = dns.ClassCHAOS

	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	status, err := nns.ServeDNS(context.TODO(), rec, req)
	require.NoError(t, err)
	require.Equal(t, dns.RcodeSuccess, status)

	res := rec.Msg.Answer[0].(*dns.TXT).Txt[0]
	require.Equal(t, "0e99bef139732856362899310a9bac1211f72d06", res)
}

func TestMapping(t *testing.T) {
	for _, tc := range []struct {
		dnsDomain string
		nnsDomain string
		request   string
		expected  string
	}{
		{
			dnsDomain: ".",
			nnsDomain: "",
			request:   "test.neofs",
			expected:  "test.neofs",
		},
		{
			dnsDomain: ".",
			nnsDomain: "",
			request:   "test.neofs.",
			expected:  "test.neofs",
		},
		{
			dnsDomain: ".",
			nnsDomain: "container.",
			request:   "test.neofs",
			expected:  "test.neofs.container",
		},
		{
			dnsDomain: ".",
			nnsDomain: ".container",
			request:   "test.neofs.",
			expected:  "test.neofs.container",
		},
		{
			dnsDomain: "containers.testnet.fs.neo.org.",
			nnsDomain: "container",
			request:   "containers.testnet.fs.neo.org",
			expected:  "container",
		},
		{
			dnsDomain: ".containers.testnet.fs.neo.org",
			nnsDomain: "container",
			request:   "containers.testnet.fs.neo.org.",
			expected:  "container",
		},
		{
			dnsDomain: "containers.testnet.fs.neo.org.",
			nnsDomain: "container",
			request:   "nicename.containers.testnet.fs.neo.org",
			expected:  "nicename.container",
		},
	} {
		nns := &NNS{}
		nns.setDNSDomain(tc.dnsDomain)
		nns.setNNSDomain(tc.nnsDomain)

		res := nns.prepareName(tc.request)
		require.Equal(t, tc.expected, res)
	}
}
