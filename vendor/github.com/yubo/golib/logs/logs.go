/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logs

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/yubo/golib/util/wait"
	"k8s.io/klog/v2"
)

const logFlushFreqFlagName = "log-flush-frequency"

var (
	logFlushFreq = pflag.Duration(logFlushFreqFlagName, 5*time.Second, "Maximum number of seconds between log flushes")
	FlagSet      = flag.NewFlagSet("klog", flag.ExitOnError)
)

func init() {
	klog.InitFlags(FlagSet)
}

// AddFlags registers this package's flags on arbitrary FlagSets, such that they point to the
// same value as the global flags.
func AddFlags(fs *pflag.FlagSet) {
	fs.AddFlag(pflag.Lookup(logFlushFreqFlagName))
}

// KlogWriter serves as a bridge between the standard log package and the glog package.
type KlogWriter struct{}

// Write implements the io.Writer interface.
func (writer KlogWriter) Write(data []byte) (n int, err error) {
	klog.InfoDepth(1, string(data))
	return len(data), nil
}

// wordSepNormalizeFunc changes all flags that contain "_" separators
func wordSepNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	return pflag.NormalizedName(strings.ReplaceAll(name, "_", "-"))
}

// InitLogs initializes logs the way we want for kubernetes.
func InitLogs() {
	pflag.CommandLine.SetNormalizeFunc(wordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(FlagSet)

	log.SetOutput(KlogWriter{})
	log.SetFlags(0)
	// The default glog flush interval is 5 seconds.
	go wait.Forever(klog.Flush, *logFlushFreq)

}

// FlushLogs flushes logs immediately.
func FlushLogs() {
	klog.Flush()
}

// NewLogger creates a new log.Logger which sends logs to klog.Info.
func NewLogger(prefix string) *log.Logger {
	return log.New(KlogWriter{}, prefix, 0)
}

// GlogSetter is a setter to set glog level.
func GlogSetter(val string) (string, error) {
	var level klog.Level
	if err := level.Set(val); err != nil {
		return "", fmt.Errorf("failed set klog.logging.verbosity %s: %v", val, err)
	}
	return fmt.Sprintf("successfully set klog.logging.verbosity to %s", val), nil
}
