package main

import (
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/cloudflare/cfssl/cli"
	"github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/csr"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type etcdConfig struct {
	CertificateRequest `yaml:",inline"`
	Etcd               EtcdConfig  `yaml:"etcd"`
	CA                 CaConfig    `yaml:"ca"`
	opts               *options    `yaml:"-"`
	c                  *cli.Config `yaml:"-"`
}

type EtcdConfig struct {
	Dir          string   `yaml:"dir"`
	Hosts        []string `yaml:"hosts"`
	ClientExpiry string   `yaml:"clientExpiry"`
	ServerExpiry string   `yaml:"serverExpiry"`
}

/*
client
bin/cfssl gencert -ca=root/ca.pem -ca-key=root/ca-key.pem \
    -config=config/config-profiles.json \
    -profile=client config/config-etcd-client.json
*/

func newEtcdCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "etcd",
		Short: "etcd",
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := ioutil.ReadFile(opts.configFile)
			if err != nil {
				return err
			}

			req := csr.CertificateRequest{
				KeyRequest: csr.NewKeyRequest(),
			}

			cf := &etcdConfig{opts: opts}
			if err := yaml.Unmarshal(b, cf); err != nil {
				return err
			}

			if cf.c, err = genEtcdConfig(cf); err != nil {
				return err
			}

			if err := etcdServer(cf, &req); err != nil {
				return err
			}

			if err := etcdPeer(cf, &req); err != nil {
				return err
			}

			if err := etcdClient(cf, &req); err != nil {
				return err
			}

			return nil
		},
	}
}

func genEtcdConfig(cf *etcdConfig) (*cli.Config, error) {

	clientExp, err := time.ParseDuration(cf.Etcd.ClientExpiry)
	if err != nil {
		return nil, err
	}

	serverExp, err := time.ParseDuration(cf.Etcd.ServerExpiry)
	if err != nil {
		return nil, err
	}

	return &cli.Config{
		CAFile:    filepath.Join(cf.CA.Dir, "ca.pem"),
		CAKeyFile: filepath.Join(cf.CA.Dir, "ca-key.pem"),
		CFG: &config.Config{
			Signing: &config.Signing{
				Profiles: map[string]*config.SigningProfile{
					"server": &config.SigningProfile{
						Expiry: serverExp,
						Usage: []string{
							"signing",
							"key encipherment",
							"client auth",
							"server auth",
						},
					},
					"peer": &config.SigningProfile{
						Expiry: serverExp,
						Usage: []string{
							"signing",
							"key encipherment",
							"server auth",
							"client auth",
						},
					},
					"client": &config.SigningProfile{
						Expiry: clientExp,
						Usage: []string{
							"signing",
							"key encipherment",
							"client auth",
						},
					},
				},
				Default: config.DefaultConfig(),
			},
		},
	}, nil
}

func etcdServer(cf *etcdConfig, req *csr.CertificateRequest) error {
	cf.c.Profile = "server"
	req.Hosts = cf.Etcd.Hosts

	if len(req.Names) == 1 {
		req.Names[0].OU = "etcd server"
	}

	return genCert(*cf.c, req, cf.Etcd.Dir, "server", cf.opts.force)
}

func etcdPeer(cf *etcdConfig, req *csr.CertificateRequest) error {
	cf.c.Profile = "peer"
	req.Hosts = cf.Etcd.Hosts

	if len(req.Names) == 1 {
		req.Names[0].OU = "etcd peer"
	}

	return genCert(*cf.c, req, cf.Etcd.Dir, "peer", cf.opts.force)
}

func etcdClient(cf *etcdConfig, req *csr.CertificateRequest) error {
	cf.c.Profile = "client"
	req.Hosts = nil

	if len(req.Names) == 1 {
		req.Names[0].OU = "etcd client"
	}

	return genCert(*cf.c, req, cf.Etcd.Dir, "client", cf.opts.force)

}
