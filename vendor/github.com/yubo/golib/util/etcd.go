/*
 * export ETCDCTL_API=3
 * etcdctl get --prefix /openfalcon
 */
package util

/*

import (
	"errors"
	"time"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/pkg/transport"
	"golang.org/x/net/context"
	"k8s.io/klog/v2"
)

const (
	ETCD_CLIENT_MAX_RETRY       = 2
	ETCD_CLIENT_REQUEST_TIMEOUT = 3 * time.Second
)

var (
	ErrNoClient = errors.New("etcdclient: etcd client is nil")
)

// just for falcon-lite(graph/transfer)
type EtcdCliConfig struct {
	Disable     bool     `json:"disable"`
	ConnTimeout int64    `json:"conn_timeout"`
	CallTimeout int64    `json:"call_timeout"`
	Endpoints   []string `json:"endpoints"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	CertFile    string   `json:"cert_file"`
	KeyFile     string   `json:"key_file"`
	CaFile      string   `json:"ca_file"`
	LeaseKey    string   `json:"lease_key"`
	LeaseValue  string   `json:"lease_value"`
	LeaseTtl    int64    `json:"lease_ttl"`
}

type EtcdCli struct {
	Conf *EtcdCliConfig

	// runtime
	leaseid     clientv3.LeaseID
	config      clientv3.Config
	client      *clientv3.Client
	ctx         context.Context
	cancel      context.CancelFunc
	connTimeout time.Duration
	callTimeout time.Duration
}

func NewEtcdCli(c *EtcdCliConfig) *EtcdCli {
	return &EtcdCli{Conf: c}
}

func (p *EtcdCli) Prestart() {
	c := p.Conf

	if c.Disable {
		return
	}

	if len(c.Endpoints) == 0 || c.Endpoints[0] == "" {
		return
	}

	p.connTimeout = time.Duration(c.ConnTimeout) * time.Second
	p.callTimeout = time.Duration(c.CallTimeout) * time.Second

	p.config = clientv3.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: p.connTimeout,
		Username:    c.Username,
		Password:    c.Password,
	}

	if c.CertFile != "" && c.KeyFile != "" {
		tlsInfo := transport.TLSInfo{
			CertFile:      c.CertFile,
			KeyFile:       c.KeyFile,
			TrustedCAFile: c.CaFile,
		}
		tlsConfig, err := tlsInfo.ClientConfig()
		if err != nil {
			klog.V(3).Infof("etcd ClientConfig() error %s", err.Error())
			return
		}
		p.config.TLS = tlsConfig
	}

	return
}

func (p *EtcdCli) keepalive() (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	if resp, err := p.client.Grant(p.ctx, p.Conf.LeaseTtl); err != nil {
		klog.V(3).Infof("etcd Grant() error %s", err.Error())
		return nil, err
	} else {
		p.leaseid = resp.ID
	}

	ctx, _ := context.WithTimeout(context.Background(), p.callTimeout)
	if _, err := p.client.Put(ctx, p.Conf.LeaseKey, p.Conf.LeaseValue,
		clientv3.WithLease(p.leaseid)); err != nil {
		klog.V(3).Infof("etcd put with lease error %s", err.Error())
		return nil, err
	}

	return p.client.KeepAlive(p.ctx, p.leaseid)
}

func (p *EtcdCli) reconnection() (err error) {
	if p.client != nil {
		p.client.Close()
	}
	p.client, err = clientv3.New(p.config)
	return err
}

func (p *EtcdCli) Start() error {
	var (
		respCh <-chan *clientv3.LeaseKeepAliveResponse
		err    error
	)

	if p.Conf.Disable {
		klog.V(3).Infof("etcd client disabled")
		return nil
	}

	p.ctx, p.cancel = context.WithCancel(context.Background())

	if err = p.reconnection(); err != nil {
		return err
	}

	if respCh, err = p.keepalive(); err != nil {
		return err
	}

	// the key will be kept forever
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case resp := <-respCh:
				if resp != nil {
					klog.V(3).Infof("etcd keepalive response %#v", resp)
					continue
				}

				if err := p.reconnection(); err != nil {
					klog.Errorf("etcd reconnection err %s", err.Error())
					time.Sleep(time.Duration(p.Conf.LeaseTtl/3) * time.Second)
					continue
				}

				if ch, err := p.keepalive(); err != nil {
					klog.Errorf("etcd keepalived err %s", err.Error())
					time.Sleep(time.Duration(p.Conf.LeaseTtl/3) * time.Second)
					continue
				} else {
					respCh = ch
				}
			}
		}
	}()
	klog.V(3).Infof("etcd keepalive success")
	return nil
}

func (p *EtcdCli) Stop() {
	if p.client == nil {
		return
	}

	// cancel background routine
	p.cancel()

	// will closed by ctx
	//p.client.Revoke(context.Background(), p.leaseid)

	p.client.Close()
	p.client = nil
}

func (p *EtcdCli) Reload(c *EtcdCliConfig) {
	p.Stop()
	p.Prestart()
	p.Start()
	return
}

func (p *EtcdCli) Get(key string) (string, error) {
	if p.client == nil {
		return "", ErrNoClient
	}

	ctx, _ := context.WithTimeout(context.Background(), p.callTimeout)
	resp, err := p.client.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) > 0 {
		return string(resp.Kvs[0].Value), nil
	}
	return "", ErrEmpty
}

func (p *EtcdCli) GetPrefix(key string) (*clientv3.GetResponse, error) {
	if p.client == nil {
		return nil, ErrNoClient
	}

	ctx, _ := context.WithTimeout(context.Background(), p.callTimeout)
	return p.client.Get(ctx, key, clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
}

func (p *EtcdCli) Put(key, value string) (err error) {
	if p.client == nil {
		return ErrNoClient
	}

	ctx, _ := context.WithTimeout(context.Background(), p.callTimeout)
	_, err = p.client.Put(ctx, key, value)
	return
}

func (p *EtcdCli) Puts(kvs map[string]string) (err error) {
	if p.client == nil {
		return ErrNoClient
	}

	i := 0
	ops := make([]clientv3.Op, len(kvs))
	for k, v := range kvs {
		ops[i] = clientv3.OpPut(k, v)
		i++
	}

	ctx, _ := context.WithTimeout(context.Background(), p.callTimeout)
	_, err = clientv3.NewKV(p.client).Txn(ctx).If().Then(ops...).Commit()
	return
}
*/
