/*
 * Copyright 2018 yubo. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */
package core

import (
	"io/ioutil"
	"testing"

	"github.com/golang/glog"
)

// get config  ParseConfigFile(values, config.yml)
// merge baseconf, config

var (
	testBaseYaml string
)

func init() {
	b, err := ioutil.ReadFile("./test/base.yml")
	if err != nil {
		panic(err)
	}
	testBaseYaml = string(b)
}

func TestConfig(t *testing.T) {
	config, err := NewConfiger("./test/conf.yml", testBaseYaml, []string{"./test/values.yml"})
	if err != nil {
		t.Error(t)
	}

	err = config.Parse()
	if err != nil {
		t.Error(t)
	}

	glog.V(3).Infof("%s", config)
}

func TestRaw(t *testing.T) {
	config, _ := NewConfiger("./test/conf.yml", testBaseYaml,
		[]string{"./test/values.yml"})
	config.Parse()

	var cases = []struct {
		path string
		want interface{}
	}{
		{"foo1", "b_bar1"},
		{"foo2", "v_bar2"},
		{"foo3", "b_bar3"},
		{"fooo.foo", "bar"},
		{"na", nil},
		{"na.na", nil},
	}

	for _, c := range cases {
		if got := config.GetRaw(c.path); got != c.want {
			t.Errorf("config.GetRaw(%s) expected %#v got %#v", c.path, c.want, got)
		}
	}
}

func TestRead(t *testing.T) {
	config, _ := NewConfiger("./test/conf.yml", testBaseYaml,
		[]string{"./test/values.yml"})
	config.Parse()

	var (
		got  []string
		path = "fooo.foos"
	)

	if err := config.Read(path, &got); err != nil {
		t.Error(err)
	} else {
		t.Logf("config.Read(%s) got %#v", path, got)
	}
}

func TestRawType(t *testing.T) {
	yml := `
ctrl:
  auth:
    google:
      client_id: "781171109477-10tu51e8bs1s677na46oct6hdefpntpu.apps.googleusercontent.com"
      client_secret: xpEoBFqkmI3KVN9pHt2VW-eN
      redirect_url: http://auth.dev.pt.xiaomi.com/v1.0/auth/callback/google
`

	var cases = []struct {
		path string
		want string
	}{
		{"ctrl", "map[string]interface {}"},
		{"ctrl.auth", "map[string]interface {}"},
		{"ctrl.auth.google", "map[string]interface {}"},
		{"ctrl.auth.google.client_id", "string"},
	}

	conf, _ := newConfiger([]byte(yml))

	for _, c := range cases {
		if got := GetType(conf.GetRaw(c.path)); got != c.want {
			t.Fatalf("GetType(conf.GetRaw(%s)) got %s want %s",
				c.path, got, c.want)
		}
	}

	// test to configer
}

func TestToConfiger(t *testing.T) {
	yml := `
ctrl:
  auth:
    google:
      client_id: "781171109477-10tu51e8bs1s677na46oct6hdefpntpu.apps.googleusercontent.com"
      client_secret: xpEoBFqkmI3KVN9pHt2VW-eN
      redirect_url: http://auth.dev.pt.xiaomi.com/v1.0/auth/callback/google
`
	var cases = []struct {
		path1 string
		path2 string
		want  string
	}{
		{"ctrl.auth", "google", "map[string]interface {}"},
		{"ctrl.auth", "google.client_secret", "string"},
		{"ctrl.auth.google", "client_id", "string"},
	}

	conf, _ := newConfiger([]byte(yml))
	for _, c := range cases {
		cf := ToConfiger(conf.GetRaw(c.path1))
		if cf == nil {
			t.Fatalf("get %s error", c.path1)
		}

		if got := GetType(cf.GetRaw(c.path2)); got != c.want {
			t.Fatalf("GetType(ToConfiger(conf.GetRaw(%s)).GetRaw(%s)) got %s want %s",
				c.path1, c.path2, got, c.want)
		} else {
			glog.V(3).Infof("%s %s got value %v", c.path1, c.path2, cf.GetRaw(c.path2))
		}

	}

}
