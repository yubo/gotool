# Mysqldiff

## Install

```
go install github.com/yubo/gotool/vimdiff@latest
```

## Usage
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
