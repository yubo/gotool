package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yubo/golib/logs"
)

// usage: mysqldiff --dsn1="root:1234@tcp(localhost:3306)/src_db?charset=utf8" --dsn2="root:1234@tcp(localhost:3306)/src_db?charset=utf8"

type Config struct {
	srcDsn string
	dstDsn string
	exec   bool
}

func main() {
	logs.InitLogs()

	cf := &Config{}
	var rootCmd = &cobra.Command{
		Use:   "mysqldiff",
		Short: "mysqldiff is a tool which compares tow MySQL databases",
		Long:  `mysqldiff is a front-end which compares the data structures (i.e. schema / table definitions) of two MySQL databases, and returns the differences as a sequence of MySQL commands suitable for piping into mysql which will transform the structure of the first database to be identical to that of the second database (c.f. diff and patch).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mysqldiff(cf)
		},
	}

	fs := rootCmd.PersistentFlags()
	fs.StringVar(&cf.srcDsn, "dsn1", "", "dsn e.g. root:1234@tcp(localhost:3306)/src_db?charset=utf8")
	fs.StringVar(&cf.dstDsn, "dsn2", "", "dsn e.g. root:1234@tcp(localhost:3306)/dst_db?charset=utf8")
	fs.BoolVar(&cf.exec, "exec", false, "exec diff sql")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
