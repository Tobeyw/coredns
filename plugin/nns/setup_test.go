package nns

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/coredns/caddy"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testImage = "nspccdev/neofs-aio-testcontainer:0.26.1"

func TestIntegration(t *testing.T) {
	//ctx, cancel := context.WithCancel(context.Background())
	//container := createDockerContainer(ctx, t, testImage)
	//defer container.Terminate(ctx)

	c := caddy.NewTestController("dns", "nns http://seed1t5.neo.org:20332 b611cdc5d9a392f947e1f333c010aebdc9f16b80")
	err := setup(c) //0x711fd7e746c635c2db8497c0af99b0770a7fe2bf   bfe27f0a77b099afc09784dbc235c646e7d71f71
	fmt.Println(err)
	require.NoError(t, err)
	//cancel()
}

func TestParseArgs(t *testing.T) {
	for _, tc := range []struct {
		args  string
		valid bool
	}{
		//{args: "", valid: false},
		//{args: "localhost", valid: false},
		//{args: "localhost:30333", valid: false},
		//{args: "http://localhost", valid: false},
		//{args: "http://localhost:30333", valid: true},
		{args: "http://seed1t5.neo.org:20332 b611cdc5d9a392f947e1f333c010aebdc9f16b80", valid: true},
		//{args: "http://localhost:30333 domain third", valid: false},
	} {
		c := caddy.NewTestController("dns", "nns "+tc.args)
		re, err := parseArgs(c)
		df := re.ContractHash.String()
		fmt.Println(df)
		var endpoint, domain string
		if re.Endpoint != "" {
			endpoint = re.Endpoint
		}
		if re.Domain != "" {
			domain = re.Domain
		}

		//contractHash := re.ContractHash.String()

		if tc.valid {
			if err != nil {
				t.Fatalf("Expected no errors, but got: %v", err)
			} else {
				res := strings.TrimSpace(endpoint + " " + domain)
				require.Equal(t, tc.args, res)
			}
		} else if !tc.valid && err == nil {
			t.Fatalf("Expected error but got nil, args: '%s'", tc.args)
		}
	}
}

func createDockerContainer(ctx context.Context, t *testing.T, image string) testcontainers.Container {
	req := testcontainers.ContainerRequest{
		Image:       image,
		WaitingFor:  wait.NewLogStrategy("aio container started").WithStartupTimeout(30 * time.Second),
		Name:        "coredns-aio",
		Hostname:    "coredns-aio",
		NetworkMode: "host",
	}
	aioC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	return aioC
}
