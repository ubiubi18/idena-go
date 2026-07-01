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

func TestMakeConfigRejectsServerCapableIpfsRoutingByDefault(t *testing.T) {
	ctx := newTestContext(t)
	require.NoError(t, ctx.Set(IpfsRoutingFlag.Name, IpfsRoutingDht))

	_, err := MakeConfig(ctx, func(cfg *Config) {})

	require.ErrorContains(t, err, `IPFS routing mode "dht" is unsafe or ambiguous`)
}

func TestMakeConfigAllowsServerCapableIpfsRoutingWithOptIn(t *testing.T) {
	t.Setenv(AllowUnsafeIpfsRoutingEnv, ipfsUnsafeRoutingEnabled)
	ctx := newTestContext(t)
	require.NoError(t, ctx.Set(IpfsRoutingFlag.Name, IpfsRoutingDht))

	cfg, err := MakeConfig(ctx, func(cfg *Config) {})

	require.NoError(t, err)
	require.Equal(t, IpfsRoutingDht, cfg.IpfsConf.Routing)
}

func TestValidateIpfsRoutingAllowsSafeKuboRoutingModes(t *testing.T) {
	for _, routing := range []string{
		"",
		IpfsRoutingAutoClient,
		IpfsRoutingDelegated,
		IpfsRoutingDhtClient,
		IpfsRoutingNone,
	} {
		require.NoError(t, validateIpfsRouting(routing))
	}
}

func TestValidateIpfsRoutingRejectsServerCapableModesByDefault(t *testing.T) {
	for _, routing := range []string{
		IpfsRoutingAuto,
		IpfsRoutingCustom,
		IpfsRoutingDht,
		IpfsRoutingDhtServer,
	} {
		require.ErrorContains(t, validateIpfsRouting(routing), "unsafe or ambiguous")
	}
}

func TestValidateIpfsRoutingAllowsServerCapableModesWithOptIn(t *testing.T) {
	t.Setenv(AllowUnsafeIpfsRoutingEnv, ipfsUnsafeRoutingEnabled)

	for _, routing := range []string{
		IpfsRoutingAuto,
		IpfsRoutingCustom,
		IpfsRoutingDht,
		IpfsRoutingDhtServer,
	} {
		require.NoError(t, validateIpfsRouting(routing))
	}
}

func TestValidateIpfsRoutingAllowsLegacyDhtServerOptIn(t *testing.T) {
	t.Setenv(AllowIpfsDhtServerEnv, ipfsUnsafeRoutingEnabled)

	require.NoError(t, validateIpfsRouting(IpfsRoutingDht))
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
