# Wickproxy
Wickproxy is a light but secure HTTP(s)、HTTP2 Proxy in Golong. Wickproxy is created by Layton and is under `MIT License`.

## Features
* A Proxy for HTTP, HTTPS or HTTP2. No need a client and easy to use.
* Highly concealed and Anti-detection. 
    * Rewrite any illegal requests to a backend server. 
    * To camouflage as a `Nginx` servers.
* Access control list. Allow or deny by IP,port, domain name or CIDR.
* Build for almost all platforms.
* Multi-users.
* High preformance.
* SSL(TLS) support.
* No need to write a configuration file.

## Build

Prerequisites:
* `Golang` 1.12 or above
* `Git` to clone this repository

To build using `go` command:
```
git clone https://github.com/ertuil/wickproxy
go build -o build/wickporxy .
```

Or to use `Make`:
```
make <platform>       # to build for special platform. Including: linux-amd64, linux-arm64 , darwin-amd64, windows-x64, windows-x86 and freebsd-amd64
make all        # to build for all three platforms
```

## Usage

### Run the server

```
wickproxy run
wickproxy -d run # run in debug mode
wickproxy -c /etc/config.json run # specify the configuration file
```

It is able for Wickproxy to reload configuration file or run in the background:
```
wickproxy start # start a wickproxy service in the background
wickproxy stop # kill the wickproxy service
wickproxy reload # ask wickproxy to reload configuration file 
```

### Basic configurations

Use `wickproxy init` to create a new copy of configuration file and use `wickproxy show` to print the configuratons.

Use `wickproxy set [key] [value]` command to change. These keys are available:

* server:       The address wickproxy will listen to. Default is `127.0.0.1:7890`
* logging:      Wickproxy will write log into `logging` file. It is only enabled in `run` or `start` command.
* timeout:      Wickproxy will try to wait `timeout` seconds when connect to the next hop.
* secure_url:   It is a probe resistance method. When `secure_url` is set, requests to `secure_url` will trick a 407 proxy-authentication response. It is used to hide as a proxy server. Example: `whtql.secure.com`.
* whitelist_url: If probe resistance is enabled, some requests may causes problems( such as `git` or `pip`), requests to `whitelist_url` will also trick a 407 proxy-authentication response for compatibility considerations. Example: `"github.com gitlab.com google.com"`.
* fallback_url: When `fallback_url` is not none, wickproxy will rewrite all illegal requests to `fallback_url`. Wickproxy will camouflage itself as a nginx server when `fallback_url` is none.
* tls_cert:     To specify tls certificate. Wickproxy will use `TLS` when `tls_cert` and `tls_key` are not none.
* tls_key:      To specify tls private key. Wickproxy will use `TLS` when `tls_cert` and `tls_key` are not none.

### User configuration file

Use `wickproxy user-add` to create a new user:
```
wickproxy user-add hustcw password1 # create a user
```

Or use `wickproxy user-del` to deleta a user:
```
wickproxy user-del hustcw
wickproxy user-del -a # delete all users
```

### Access Control List

Use `wickproxy acl-list` to print the rules with number. Use `wickproxy acl-add` to insert a new rule:
```
wickproxy -c config.json acl-list
wickproxy acl-add twitter.com deny
wickproxy acl-add --index=0 google.com allow
wickproxy acl-add 1.1.1.1 allow
wickproxy acl-add 192.168.3.0/24 deny
wickproxy acl-add 10.10.10.10:445 allow
```

Use `wickproxy acl-del` to delete a rule in ACL:
```
wickproxy acl-del 1 # delete rule number 1
wickproxy acl-del -a # clear the ACL.
```

### Help

```
wickproxy version # get version and platform information
wickproxy help    # get help manual
```