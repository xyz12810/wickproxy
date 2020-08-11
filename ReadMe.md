# Wickproxy
Wickproxy is a lightweight but security HTTP, HTTPS, and  HTTP2 Proxy with detection. Wickproxy is written in Golang and is under `MIT License`.

## Features
* A Proxy for HTTP, HTTPS, and HTTP2. Unlike other proxies, Wickproxy is an HTTP proxy and no client is needed.
* Highly concealed and Anti-detection. 
    * Rewrite any illegal requests to a backend server. 
    * To camouflage as a `Nginx` servers.
    * Wickproxy can work as a frontend server application before `Caddy` or `Nginx`.
* Access control list. Allow or deny by IP, ports, domain name, or CIDR.
* Build for almost all platforms. Wickproxy is compiled for Windows, OS X, Linux, and Freebsd.
* No need to write a configuration file. It can be used out of the box.
* Multi-users.
* High performance.

### Anti-Detection
In HTTP(s) proxy, it is common that no authentication information is sent in the first packet, then the server should return a `407 Proxy-Authenticate` to indicate that the authentication information should be sent. However, this behavior exposes the fact of it is a proxy server.

In order to anti-detection, only requests to `secure_host`(such as `pr.wickproxy.org` will trick a `407 Proxy-Authenticate` response and other requests will be resent to backend servers or response with a `404 Not Found` error.

However, it is nay not be compatible with some software such as `git` command. A `whitelist_hosts` is introduced to solve this problem. Hosts in `whitelist_hosts` will also trick a `407 Proxy-Authenticate`.

### Fallback Model
It is easy to use Wickproxy as the frontend server listening on port 443 or 80. Any invalid requests will be sent to `fallback`. Then, an Nginx or caddy server listen on fallback. If no `fallback` is set, Wickproxy will 
Pretend to be an Nginx server.

## Build
Prerequisites:
* `Golang` 1.12 or above
* `git` to clone this repository

It is easy to build Wickproxy using `go` command:
```
git clone https://github.com/ertuil/wickproxy
go build -o build/wickporxy .
```

Another way to compile Wickproxy is to use `Make` command:
```
make <platform>       # to build for special platform. Including: linux-amd64, linux-arm64 , darwin-amd64, windows-x64, windows-x86 and freebsd-amd64
make all        # to build for all three platforms
```

## Usage
### Get Help
```
wickproxy version # get version and platform information
wickproxy help    # get help manual
```

### Run the server
Before your first run, it is necessary to use `wickproxy init` to create a new copy of the configuration file.
```
wickproxy init
```

Now, it is able to run Wickproxy using `wickproxy run`. You can also use `-d` to enable debug mode or `-c` to specify a configuration file:
```
wickproxy run
wickproxy -d run # run in debug mode
wickproxy -c /etc/config.json run # specify the configuration file
```

Wickproxy is able to run in the background as a server and is able for Wickproxy to reload the configuration file without a restart.
```
wickproxy start 	# start a wickproxy service in the background
wickproxy stop 	# kill the wickproxy service
wickproxy reload	# ask wickproxy to reload configuration file 
```

### Basic configurations
It is designed to be easy to configure. Use `wickproxy show` to print the current configuration. A demo of the configuration file is located in `doc/example-config.json`.

Use `wickproxy set [key] [value]` command to write settings to the configuration file. These keys are available:

* server:       The address Wickproxy will listen at. Default address is `127.0.0.1:7890`.
* logging:      Wickproxy will write logs into the `logging` file. It is only enabled in `run` or `start` command.
* timeout:      Wickproxy will try to wait `timeout` seconds when connect to the next hop.
* secure_host:   It is one of the probe resistance methods. When `secure_host` is set, only requests to `secure_url` will trick a `407 proxy-authentication` response. It is used to hide the fact that Wickproxy is a proxy server. Example: `whtql.secure.com`.
* whitelist_hosts: If probe resistance is enabled, some requests may causes problems( such as `git` or `pip`), requests to `whitelist_hosts` will also trick a 407 proxy-authentication response for compatibility considerations. Example: `"github.com gitlab.com google.com"`.
* fallback: When `fallback` is not none, Wickproxy will rewrite all invalid requests to `fallback`. Wickproxy will pretend to be an Nginx server when `fallback` is not set.
* tls_cert:     To specify TLS certificate. Wickproxy will enable HTTPS, HTTP2 mode when both `tls_cert` and `tls_key` are not none. Otherwise, Wickproxy will work as an HTTP server.
* tls_key:      To specify TLS private key.

### User configure
To manage the users, several commands are used to add, list, and delete users. Use `wickproxy user-add` command to sign up a new user:
```
wickproxy user-add <username> <password> # create a user
```

Use `wickproxy user-del` to delete a user:
```
wickproxy user-del <username>	# Delete user by his username
wickproxy user-del -a 			# Delete all exist users
```

### Access Control List
Wickproxy is able to deny or accept some proxy requests using ACL. It is a quite common scene to deny requests to some ports (in order to prevent P2P usage).

Use the `wickproxy acl-list` to print the rules with numbered. Use `wickproxy acl-add` to insert a new rule and `wickproxy acl-del` to delete a rule. Here are some examples:

```
# List ACL in the configuration file `config.json`
wickproxy -c config.json acl-list
# Insert a rule that accepts requests to `twitter.com`
wickproxy acl-add twitter.com deny
# Insert a rule as the first rule in the ACL that allows requests to `google.com`
wickproxy acl-add --index=0 google.com allow
# Insert a rule to allow requests to 1.1.1.1
wickproxy acl-add 1.1.1.1 allow
# Insert a rule to deny all requests to CIDR 192.168.3.0/24
wickproxy acl-add 192.168.3.0/24 deny
# Insert a rule to allow requests to 10.10.10.10:445
wickproxy acl-add 10.10.10.10:445 allow
# Delete the rule number 1
wickproxy acl-del 1
# Clear all rules in the ACL
wickproxy acl-del -a
```