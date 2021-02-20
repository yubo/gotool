# Mysqldiff

## install

```
go get -u github.com/yubo/gotool/vimdiff
```

e.g.

```shell
export ORIG_DIR=/tmp/a
export CUR_DIR=/tmp/b
mkdir -p /tmp/{a,b}
echo 'a' > ${ORIG_DIR}/file
echo 'b' > ${CUR_DIR}/file

cd ${CUR_DIR}
vimdiff ./file
```
