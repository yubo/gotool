package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/cloudflare/cfssl/cli"
	"github.com/cloudflare/cfssl/cli/genkey"
	"github.com/cloudflare/cfssl/cli/sign"
	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/signer"
)

func fileExist(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return true
}

func checkDir(dir string) {
	_, err := os.Stat(dir)
	if err != nil && os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
}

type outputFile struct {
	Filename string
	Contents string
	IsBinary bool
	Perms    os.FileMode
}

func writeCert(baseName string, force bool, key, csrBytes, cert []byte) error {
	var outs []outputFile

	if len(key) > 0 {
		outs = append(outs, outputFile{
			Filename: baseName + "-key.pem",
			Contents: string(key),
			Perms:    0600,
		})
	}

	if len(csrBytes) > 0 {
		outs = append(outs, outputFile{
			Filename: baseName + ".csr",
			Contents: string(cert),
			Perms:    0664,
		})
	}

	if len(cert) > 0 {
		outs = append(outs, outputFile{
			Filename: baseName + ".pem",
			Contents: string(cert),
			Perms:    0664,
		})
	}

	if !force {
		for _, e := range outs {
			if fileExist(e.Filename) {
				return fmt.Errorf("file %s is exist, use -f to override it", e.Filename)
			}
		}
	}

	for _, e := range outs {
		writeFile(e.Filename, e.Contents, e.Perms)
	}

	return nil
}

func writeFile(filespec, contents string, perms os.FileMode) {
	err := ioutil.WriteFile(filespec, []byte(contents), perms)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully created %s\n", filespec)
}

func genCert(c cli.Config, req *csr.CertificateRequest, dir, baseName string, force bool) error {
	var key, csrBytes []byte
	var err error
	g := &csr.Generator{Validator: genkey.Validator}
	csrBytes, key, err = g.ProcessRequest(req)
	if err != nil {
		key = nil
		return err
	}

	s, err := sign.SignerFromConfig(c)
	if err != nil {
		return err
	}

	var cert []byte
	signReq := signer.SignRequest{
		Request: string(csrBytes),
		Hosts:   signer.SplitHosts(c.Hostname),
		Profile: c.Profile,
		Label:   c.Label,
	}

	cert, err = s.Sign(signReq)
	if err != nil {
		return err
	}

	checkDir(dir)
	return writeCert(path.Join(dir, baseName), force, key, csrBytes, cert)

}
