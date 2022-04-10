## ssl tool

#### Install
```
go install github.com/yubo/gotool/ssltool@latest
```

#### Configure
```
cat > ./ssltool.yml <<'EOF'
CN: etcd
key:
  algo: "ecdsa"
  size: 256
names:
  - C: CN
    ST: BJ
    L: BJ
    O: k8s

ca:
  dir: ./ca

etcd:
  dir: ./etcd
  clientExpiry: "876000h"
  serverExpiry: "876000h"
  hosts:
    - 127.0.0.1
    - 1.1.1.1
    - 1.1.1.2
EOF
```

#### Generate CA
```
ssltool ca --conf ./ssltool.yml
```

#### Generate ETCD Certs
```
ssltool etcd --conf ./ssltool.yml
```
