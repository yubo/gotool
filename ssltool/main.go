package main

import (
	"fmt"
	"os"

	"github.com/cloudflare/cfssl/csr"
	"github.com/spf13/cobra"
)

type options struct {
	configFile string
	keyFile    string
	force      bool
}

type CertificateRequest struct {
	CN         string          `json:"CN" yaml:"CN"`
	Names      []csr.Name      `json:"names" yaml:"names"`
	Hosts      []string        `json:"hosts" yaml:"hosts"`
	KeyRequest *csr.KeyRequest `json:"key,omitempty" yaml:"key,omitempty"`
}

func main() {
	opts := &options{}
	rootCmd := &cobra.Command{
		Use:          "ssltool",
		Short:        "ssltool",
		Long:         "ssltool",
		SilenceUsage: true,
	}

	fs := rootCmd.PersistentFlags()
	fs.StringVarP(&opts.configFile, "conf", "c", "./ssltool.yml", "config")
	fs.StringVarP(&opts.keyFile, "keyFile", "k", "", "key file")
	fs.BoolVarP(&opts.force, "force", "f", false, "force")

	rootCmd.AddCommand(
		newCaCmd(opts),
		newEtcdCmd(opts),
		newHttpCmd(opts),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
