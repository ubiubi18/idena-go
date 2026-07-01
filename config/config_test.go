package config

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

func TestApplyProfileDefaultsToDhtClientRouting(t *testing.T) {
	cfg := getDefaultConfig(DefaultDataDir)
	ctx := newTestContext(t)

	applyProfile(ctx, cfg)

	require.Equal(t, DefaultIpfsRouting, cfg.IpfsConf.Routing)
}

func TestApplyProfilePreservesConfiguredIpfsRouting(t *testing.T) {
	cfg := getDefaultConfig(DefaultDataDir)
	cfg.IpfsConf.Routing = "dht"
	ctx := newTestContext(t)

	applyProfile(ctx, cfg)

	require.Equal(t, "dht", cfg.IpfsConf.Routing)
}

func TestApplyIpfsFlagsOverridesRouting(t *testing.T) {
	cfg := getDefaultConfig(DefaultDataDir)
	ctx := newTestContext(t)

	require.NoError(t, ctx.Set(IpfsRoutingFlag.Name, IpfsRoutingDht))
	applyIpfsFlags(ctx, cfg)

	require.Equal(t, IpfsRoutingDht, cfg.IpfsConf.Routing)
}

func TestMakeConfigRejectsInvalidIpfsRouting(t *testing.T) {
	ctx := newTestContext(t)
	require.NoError(t, ctx.Set(IpfsRoutingFlag.Name, "bogus"))

	_, err := MakeConfig(ctx, func(cfg *Config) {})

	require.ErrorContains(t, err, `invalid IPFS routing mode "bogus"`)
}

func TestValidateIpfsRoutingAllowsKuboRoutingModes(t *testing.T) {
	for _, routing := range []string{
		"",
		IpfsRoutingAuto,
		IpfsRoutingAutoClient,
		IpfsRoutingCustom,
		IpfsRoutingDelegated,
		IpfsRoutingDht,
		IpfsRoutingDhtClient,
		IpfsRoutingDhtServer,
		IpfsRoutingNone,
	} {
		require.NoError(t, validateIpfsRouting(routing))
	}
}

func newTestContext(t *testing.T) *cli.Context {
	t.Helper()

	app := cli.NewApp()
	app.Flags = []cli.Flag{
		CfgFileFlag,
		DataDirFlag,
		IpfsRoutingFlag,
		ProfileFlag,
	}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, f := range app.Flags {
		f.Apply(flagSet)
	}
	return cli.NewContext(app, flagSet, nil)
}
