package main

import (
	"io/ioutil"
	"path"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

type caConfig struct {
	CertificateRequest `yaml:",inline"`
	CA                 CaConfig `yaml:"ca"`
}

type CaConfig struct {
	Dir string `yaml:"dir"`
}

// ./bin/cfssl gencert -initca config/config-ca.json | ./bin/cfssljson -bare root/ca
func newCaCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "ca",
		Short: "ca",
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := ioutil.ReadFile(opts.configFile)
			if err != nil {
				return err
			}

			req := csr.CertificateRequest{
				KeyRequest: csr.NewKeyRequest(),
			}

			cf := &caConfig{}
			if err := yaml.Unmarshal(b, cf); err != nil {
				return err
			}

			req.CN = cf.CN
			req.Names = cf.Names
			if cf.KeyRequest != nil {
				req.KeyRequest = cf.KeyRequest
			}

			var cert, csrPEM, key []byte
			if opts.keyFile != "" {
				cert, csrPEM, err = initca.NewFromPEM(&req, opts.keyFile)
			} else {
				cert, csrPEM, key, err = initca.New(&req)
				if err != nil {
					klog.Infof("req %#v, %v,  %s", req, req.KeyRequest, err)
					return err
				}
			}

			checkDir(cf.CA.Dir)
			return writeCert(path.Join(cf.CA.Dir, "ca"),
				opts.force, key, csrPEM, cert)

		},
	}
}
