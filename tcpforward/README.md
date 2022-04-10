# TcpForward

## Install

```
go install github.com/yubo/gotool/tcpforward@latest
```

## Usage

forward localhost:8080 to 192.168.1.1:80

```
tcpforward -l localhost --lport 8080 -r 192.168.1.1 --rport 80
```

