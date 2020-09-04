## install

```
go get github.com/yubo/gotool/rpmdumper
```

## example

```
$ls
nginx-1.16.1-1.module_el8.1.0+250+351caf85.x86_64.rpm

$rpm -q --scripts -p ./nginx-1.16.1-1.module_el8.1.0+250+351caf85.x86_64.rpm | rpmdumper -
warning: ./nginx-1.16.1-1.module_el8.1.0+250+351caf85.x86_64.rpm: Header V3 RSA/SHA256 Signature, key ID 8483c65d: NOKEY

$ls
nginx-1.16.1-1.module_el8.1.0+250+351caf85.x86_64.rpm  postinstall  postuninstall  preuninstall
```
