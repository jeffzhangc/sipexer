# sipexer #

Modern and flexible SIP ([RFC3261](https://datatracker.ietf.org/doc/html/rfc3261)) command line tool.

Project URL:

  * https://github.com/miconda/sipexer

**Table Of Content**

  * [Overview](#overview)
  * [Features](#features)
  * [Installation](#installation)
    + [Compile From Sources](#compile-from-sources)
    + [Download Binary Release](#download-binary-release)
  * [Usage](#usage)
    + [Examples](#examples)
  * [Wrapped CLI (`sipx`)](#wrapped-cli-sipx)
    + [Manage phone profiles](#manage-phone-profiles)
    + [Manage service addresses](#manage-service-addresses)
    + [Dial](#dial)
    + [History](#history)
  * [Target Address](#target-address)
  * [Message Template](#message-template)
    + [Template Data](#template-data)
    + [Template Fields](#template-fields)
  * [Alternatives](#alternatives)
  * [License](#license)
  * [Contributions](#contributions)

## Overview

`sipexer` is a cli tool that facilitates sending SIP requests to servers. It uses a flexible template
system to allow defining many parts of the SIP request via command line parameters. It has support
for UDP, TCP, TLS and WebSocket (WS and WSS) transport protocols, being suitable to test modern
WebRTC SIP servers.

`sipexer` is not a SIP cli softphone, but a tool for crafting SIP requests mainly for the purpose
of testing SIP signaling routing or monitoring servers. It can also simulate some common scenarios,
like: registration and call itself, call between two users or subscribe session.

It is written in Go, aiming to be usable from Linux, MacOS or Windows.

The meaning of the name `sipexer`: randomly selected to be easy to write and pronounce,
quickly after thought of it as the shortening of `SIP EXEcutoR`.

`sipexer` in action sending a SIP OPTIONS request:

![SIP OPTIONS Request](https://github.com/miconda/sipexer/raw/main/misc/img/sipexer-options.gif)

## Features

Among features:

  * send OPTIONS request (quick SIP ping to check if server is alive)
  * do registration and un-registration with customized expires value
  and contact URI
  * authentication with plain or HA1 passwords
  * hashing algorithms for authentication: MD5, SHA1, SHA256, SHA512-256, SHA512
  * set custom SIP headers
  * template system for building SIP requests
  * fields in the templates can be set via command line parameters or a JSON file
  * variables for setting field values (e.g., random number, data, time, environment
  variables, uuid, random string, ...)
  * simulate SIP calls at signaling layer (INVITE-wait-BYE)
  * option for late-offer SDP
  * respond to requests coming during SIP calls (e.g., OPTIONS keepalives)
  * send instant messages with SIP MESSAGE requests
  * color output mode for easier troubleshooting
  * support for many transport layers: IPv4 and IPv6, UDP, TCP, TLS and WebSocket (for WebRTC)
  * send SIP requests of any type (e.g., INFO, SUBSCRIBE, NOTIFY, ...)
  * simulate call to itself, by registering first, then calling its own AoR, answering
  and hanging up
  * simulate call between two users, registering both, then calling from first user
  to the second one, answering and hanging up
  * simulate subscribe session: send SUBSCRIBE and wait for NOTIFY requests
  * optionally re-resolve an FQDN target via DNS before each in-dialog request (UDP only)

## Installation

### Compile From Sources

First install [Go](http://golang.org). Once the Go environment is configured, clone `sipexer` git repository:

```
git clone https://github.com/miconda/sipexer
```

Download dependencies and build:

```
cd sipexer
go get ./...
go build .
```

The binary `sipexer` should be generated in the current directory.

The repository also ships a wrapped CLI, `sipx`, that manages phone
profiles and dial history on top of the engine. Build both with `make`
(the engine binary is untouched):

```
make              # build ./sipexer and ./sipx for this machine
make test         # go vet + go test
make dist         # cross-compile mac+linux x amd64+arm64 bundles into dist/
make install      # stage both binaries + doc under /usr/local (sudo if needed)
```

Each `dist/` bundle (`sipexer-sipx-<version>-<os>-<arch>.tar.gz`) contains:

```
bin/sipexer                    # the engine
bin/sipx                       # the wrapper (finds bin/sipexer next to it)
share/doc/sipx/SIPX.md         # standalone sipx usage guide (Chinese)
```

See **[SIPX.md](SIPX.md)** for the full `sipx` manual.

**Note:** On some OS distributions, it may be required to run the `make`
commands with `CGO_ENABLED=0`, like:

```
CGO_ENABLED=0 make
```

### Download Binary Release

Binary releases for `Linux`, `MacOS` and `Windows` are available at:

  * https://github.com/miconda/sipexer/releases

## Usage

Prototype:

```
sipexer [options] [target]
```

See `sipexer -h` for the command line options and arguments.

Defaults:
  * target address: `sip:127.0.0.1:5060`
  * SIP method: `OPTIONS`
  * From user: `alice`
  * From domain: `localhost`
  * To user: `bob`
  * To domain: `localhost`

## Wrapped CLI (`sipx`)

`sipx` is a thin wrapper around `sipexer` that keeps a set of phone
profiles (multiple registered numbers, multiple service addresses, auth
credentials) and a dial history, so repeated outbound tests become one
short command. The engine binary itself is never modified; `sipx` locates
it at runtime and runs it as a subprocess.

Engine discovery order (`sipx doctor` shows the resolved path):
  * `--engine <path>` flag (on any command)
  * `SIPEXER_BIN` environment variable
  * a `sipexer` binary in the same directory as `sipx`
  * `sipexer` on `PATH`

Configuration lives in `~/.config/sipexer/` (`profiles.json`,
`history.json`), overridable with `--config-dir`, `SIPEXER_CONFIG_DIR`,
or `$XDG_CONFIG_HOME/sipexer`. Passwords and HA1 digests are stored
obfuscated at rest — this is defense-in-depth, not encryption; protect
the directory with file permissions.

### Manage phone profiles

```
sipx phone add work --auth-user 18800012001 --auth-password '...' \
    --number 'desk=1001@example.com' --number 'mob=1002@example.com' \
    --server udp://pbx.example.com:5060 --server-default \
    --register-first --method INVITE --set-user --contact-build \
    --sw 60000 --vl 3
sipx phone list
sipx phone show work
sipx phone edit work --number 'newline=2001@example.com'
sipx phone default work
sipx phone rm work
```

Typed call defaults stored on the profile (only emitted when set):
`--sw`, `--cd`, `--rt`, `--timeout`, `--timeout-connect`,
`--timeout-write`, `--vl`, `--co`, `--com`, `--ti`, `--ex`, `--ua`,
`--cu`, `--ct`, `--mb`, `--method`, `--xh`, `--register-first`,
`--set-user`, `--contact-build`, plus `--extra-dial-flag` for anything
else. Per-dial CLI flags override the stored defaults.

### Manage service addresses

```
sipx server add work tcp://pbx2.example.com:5060 --default
sipx server list work
sipx server rm work tcp://pbx2.example.com:5060
```

### Dial

```
sipx phone default work              # mark the current profile
sipx dial work 320196120001          # raw number
sipx dial 320196120001               # uses the default profile (no name needed)
sipx dial desk                       # profile number alias (via default profile)
sipx dial @0                         # most recent history entry
sipx dial boss                       # history-entry alias
sipx dial -s udp://other:5060 1001   # override server
sipx dial work 1001 -- -vl 3         # raw engine flags after --
```

With no target and a terminal, `sipx dial` prints the profile's recent dial
history and a manual-number prompt; pick an entry, a number alias, or type a
raw number. Without a terminal it fails with guidance. `sipx phone default
[name]` sets the current profile (marked `*` in `phone list`); a single
profile is the implicit default.

**Call teardown:** after a successful INVITE the engine sends BYE when the
session wait (`--sw`, stored in the profile or passed on the command line)
expires — the clean way to hang up. `sipexer` has no signal handling, so
**Ctrl-C kills the process without a BYE** and the gateway must time the
dialog out. Avoid interrupting a dial in progress; to end a call sooner,
use a smaller `--sw`.

### History

```
sipx history                              # list (most recent first)
sipx history alias 0 boss                 # label the newest entry
sipx history clear                        # empty the store (asks first)
```

Each (profile, from-number, target) is kept as a **single** entry: re-dialing
the same number updates that entry's timestamp and moves it to the top
(its alias is preserved). The store is capped at 100 entries, oldest evicted
first.

### Examples

Send an `OPTIONS` request over `UDP` to `127.0.0.1` and port `5060` - couple of variants:

```
sipexer
sipexer 127.0.0.1
sipexer 127.0.0.1 5060
sipexer udp 127.0.0.1 5060
sipexer udp:127.0.0.1:5060
sipexer sip:127.0.0.1:5060
sipexer "sip:127.0.0.1:5060;transport=udp"
```

Specify a different R-URI:

```
sipexer -ruri sip:alice@server.com udp:127.0.0.1:5060
```

Send from UDP local port 55060:

```
sipexer -laddr 127.0.0.1:55060 udp:127.0.0.1:5060
```

Send `REGISTER` request with generated contact, expires as well as user and password authentication:

```
sipexer -register -cb -ex 600 -au alice -ap test123 udp:127.0.0.1:5060
```

Send `REGISTER` request with expires 60s, wait 20000ms (20s) and then unregister:

```
sipexer -register -vl 3 -co -com -ex 60 -fuser alice -cb -ap "abab..." -ha1 -sd -sw 20000 udp:127.0.0.1:5060
```

Set `fuser` field to `carol`:

```shell
sipexer -sd -fu "carol" udp:127.0.0.1:5060
# or 
sipexer -sd -fv "fuser:carol" udp:127.0.0.1:5060
```

Set `fuser` field to `carol` and `tuser` field to `david`, with
R-URI user same as To-user when providing proxy address destination:

```shell
sipexer -sd -fu "carol"  -tu "david" -su udp:127.0.0.1:5060
# or
sipexer -sd -fv "fuser:carol"  -fv "tuser:david" -su udp:127.0.0.1:5060
```

Add extra headers:

```
sipexer -sd -xh "X-My-Key:abcdefgh" -xh "P-Info:xyzw" udp:127.0.0.1:5060
```

Send `MESSAGE` request with body:

```
sipexer -message -mb 'Hello!' -sd -su udp:127.0.0.1:5060
```

Send `MESSAGE` request with body over `tcp`:

```
sipexer -message -mb 'Hello!' -sd -su tcp:127.0.0.1:5060
```

Send `MESSAGE` request with body over `tls`:

```
sipexer -message -mb 'Hello!' -sd -su tls:127.0.0.1:5061
```

Send `MESSAGE` request with body over `wss` (WebSocket Secure):

```
sipexer -message -mb 'Hello!' -sd -su wss://server.com:8443/sip
```

Send `MESSAGE` request with body over `ws` (WebSocket):

```
sipexer -message -mb 'Hello!' -sd -su ws://server.com:8080/sip
```

Send `INVITE` request with default From user `alice` and To user `bob`:

```
sipexer -invite -vl 3 -co -com -sd -su udp:server.com:5060
```

Initiate a call from `alice` to `bob`, with user authentication providing the
password in HA1 format, waiting 10000 milliseconds before sending the `BYE`,
with higher verbosity level (`3`) and color printing:

```shell
sipexer -invite -vl 3 -co -com -fuser alice -tuser bob -cb -ap "4a4a4a4a4a..." -ha1 -sw 10000 -sd -su udp:server.com:5060
```

Run a self-call flow (REGISTER, self-INVITE, `100/180/200`, wait ACK, then BYE):

```shell
sipexer --call-self --ring-time 2000 --call-duration 10000 -fuser alice  -ex 60 \
  -cb -sd -su udp:server.com:5060
```

Register the From user first, then send an `INVITE` on the same session:

```shell
sipexer --register-first --invite -fuser alice -tuser bob -cb -sd -su tcp:server.com:5060
```

Run a two-user call flow over UDP, TCP, TLS, WS, or WSS (user1 from regular
flags, user2 from `--ua2-*` flags). Each user keeps a separate local socket and
reuses its registration connection for the call. Use `--ua2-target` when user2
must register through a different destination; otherwise it uses user1's target:

```shell
sipexer --call-user --ring-time 2000 --call-duration 10000 \
  --fuser alice --laddr 127.0.0.1:5062 \
  --ua2-fuser bob --ua2-laddr 127.0.0.1:5064 \
  -cb -sd -su udp:server.com:5060

sipexer --call-user --ring-time 2000 --call-duration 10000 \
  --fuser alice --laddr 127.0.0.1:5062 \
  --ua2-fuser bob --ua2-laddr 127.0.0.1:5064 \
  -cb -sd -su tcp:server.com:5060

sipexer --call-user --ring-time 2000 --call-duration 10000 \
  --fuser alice --laddr 127.0.0.1:5062 \
  --ua2-fuser bob --ua2-laddr 127.0.0.1:5064 --ua2-target tcp:server2.com:5060 \
  -cb -sd -su udp:server.com:5060
```

Send `SUBSCRIBE`, then keep the session open for `--sessionwait` to receive
`NOTIFY` and auto-reply `200 OK`:

```shell
sipexer --subscribe-session --sessionwait 15000 -ex 60 -xh "Event:presence" \
  --fuser alice -cb -sd -su tcp:server.com:5060
```

To remove the default value for implicit fields (e.g., `useragent`), the `-no-val` value can
be provided (which is default `no`), like:

```shell
sipexer -ua no ...
```

Or:

```shell
sipexer -no-val skip -ua skip ...
```

## Target Address

The target address can be provided as last arguments to the `sipexer` command. It is
optional, if not provided, then the SIP message is sent over `UDP` to `127.0.0.1` port `5060`.

The format can be:

  * SIP URI (e.g., `sip:user@server.com:5080;transport=tls`)
  * SIP proxy socket address in format `proto:host:port` (e.g., `tls:server.com:5061`)
  * WS URL (e.g., `ws://server.com:8080/webrtc`)
  * WSS URL (e.g., `wss://server.com:8442/webrtc`)
  * only the server `hostname` or `IP` (e.g., `server.com`)
  * `host:port` (transport protocol is set to `UDP`)
  * `proto:host` (port is set to `5060`)
  * `host port` (transport protocol is set to `UDP`)
  * `proto host` (port is set to `5060`)
  * `proto host port` (same as `proto:host:port`)


## Message Template

### Template Data

The message to be sent via the SIP connection is built from a template file and a fields file.

The template file can contain any of the directives supported by Go package `text/template` - for more see:

  * https://golang.org/pkg/text/template/

Example:

```
{{.method}} {{.ruri}} SIP/2.0
Via: SIP/2.0/{{.viaproto}} {{.viaaddr}}{{.rport}};branch=z9hG4bKSG.{{.viabranch}}
From: {{if .fname}}"{{.fname}}" {{end}}<sip:{{if .fuser}}{{.fuser}}@{{end}}{{.fdomain}}>;tag={{.fromtag}}
To: {{if .tname}}"{{.tname}}" {{end}}<sip:{{if .tuser}}{{.tuser}}@{{end}}{{.tdomain}}>
Call-ID: {{.callid}}
CSeq: {{.cseqnum}} {{.method}}
{{if .subject}}Subject: {{.subject}}{{else}}$rmeol{{end}}
{{if .date}}Date: {{.date}}{{else}}$rmeol{{end}}
{{if .contacturi}}Contact: {{.contacturi}}{{if .contactparams}};{{.contactparams}}{{end}}{{else}}$rmeol{{end}}
{{if .expires}}Expires: {{.expires}}{{else}}$rmeol{{end}}
{{if .useragent}}User-Agent: {{.useragent}}{{else}}$rmeol{{end}}
Content-Length: 0

```

Example SDP body template:

```
v=0{{.cr}}
o={{.sdpuser}} {{.sdpsessid}} {{.sdpsessversion}} IN {{.sdpaf}} {{.localip}}{{.cr}}
s=call{{.cr}}
c=IN {{.sdpaf}} {{.localip}}{{.cr}}
t=0 0{{.cr}}
m=audio {{.sdprtpport}} RTP 0 8 101{{.cr}}
a=rtpmap:0 pcmu/8000{{.cr}}
a=rtpmap:8 pcma/8000{{.cr}}
a=rtpmap:101 telephone-event/8000{{.cr}}
a=sendrecv{{.cr}}
```

The internal templates can be found at the top of `sipexer.go` file:

  * https://github.com/miconda/sipexer/blob/main/sipexer.go

### Template Fields

The fields file has to contain a JSON document with the fields to be replaced
in the template file. The path to the JSON file is provided via `-ff` or `--fields-file`
parameters.

When the `--fields-eval` of `-fe` cli option is provided, `sipexer` evaluates the values of the
fields in the root structure of the JSON document. That means special tokens (expressions)
are replaced if the value of the field is a string matching one of the next:

  * `"$cr"` - replace with `\r`
  * `"$dateansic"` - replace with output of `time.Now().Format(time.ANSIC)`
  * `"$datefull"` - replace with output of `time.Now().String()`
  * `"$daterfc1123"` - replace with output of `time.Now().Format(time.RFC1123)`
  * `"$dateunix"` - replace with output of `time.Now().Format(time.UnixDate)`
  * `"$env(name)"` - replace with the value of the environment variable `name`
  * `"$add(name)"` - return the current value associate with `name` plus `1` (initial value is `0`)
  * `"$add(name,val)"` - return the current value associate with `name` plus `val` (initial value is `0`)
  * `"$sub(name)"` - return the current value associate with `name` minus `1` (initial value is `0`)
  * `"$sub(name,val)"` - return the current value associate with `name` minus `val` (initial value is `0`)
  * `"$mul(name,val)"` - return the current value associate with `name` multiplied with `val` (initial value is `1`)
  * `"$div(name,val)"` - return the current value associate with `name` divided by `val` (initial value is `1`)
  * `"$dec(name)"` - return the decremented value, first to return is 999999
  * `"$dec(name,val)"` - return the decremented value, first to return is`val - 1`
  * `"$inc(name)"` - return the incremented value, first to return is 1
  * `"$inc(name,val)"` - return the incremented value, first to return is`val + 1`
  * `"$randseq"` - replace with a random number from `1` to `1 000 000`
  * `"$rand(max)"` - replace with a random number from `0` to `max`
  * `"$rand(min,max)"` - replace with a random number from `min` to `max`
  * `"$randan(len)"` - random alphanumeric string of length `len`
  * `"$randan(minlen,maxlen)"` - random alphanumeric string with length from `minlen` to `maxlen`
  * `"$randhex(len)"` - random hexadecimal string of length `len`
  * `"$randhex(minlen,maxlen)"` - random hexadecimal string with length from `minlen` to `maxlen`
  * `"$randnum(len)"` - random numeric string of length `len`
  * `"$randnum(minlen,maxlen)"` - random numeric string with length from `minlen` to `maxlen`
  * `"$randstr(len)"` - random alphabetic string of length `len`
  * `"$randstr(minlen,maxlen)"` - random alphabetic string with length from `minlen` to `maxlen`
  * `"$rmeol"` - remove next end of line character `\n`
  * `"$timestamp"` - replace with output of `time.Now().Unix()` - time stamp in seconds
  * `"$timems"` - replace with output of `time.Now().UnixMilli()` - time stamp in milliseconds
  * `"$lf"` - replace with `\n`
  * `"$viabranchid"` - replace with `z9hG4bKSG.` concatenated with a UUID (universally unique
    identifier) value
  * `"$uuid"` - replace with a UUID (universally unique identifier) value
  * `"$uuidb64r"` - replace with a UUID in base64url (no padding) format, replacing `9`
  with `99`, `-` with `90`, and `_` with `91`
  * `"$uuidb64u"` - replace with a UUID in base64url (no padding) format

When internal template is used, `--fields-eval` is turned on.

Example fields file:

```json
{
	"method": "OPTIONS",
	"fuser": "alice",
	"fdomain": "localhost",
	"tuser": "bob",
	"tdomain": "localhost",
	"viabranch": "$uuid",
	"rport": ";rport",
	"fromtag": "$uuid",
	"totag": "",
	"callid": "$uuid",
	"cseqnum": "$randseq",
	"date": "$daterfc1123",
	"sdpuser": "sipexer",
	"sdpsessid": "$timestamp",
	"sdpsessversion": "$timestamp",
	"sdpaf": "IP4",
	"sdprtpport": "$rand(20000,40000)"
}
```

The internal fields data can be found at the top of `sipexer.go` file.

The values for fields can be also provided using `--field-val` or `-fv` cli parameter, in
format `name:value`, for example:

```
sipexer --field-val="domain:openrcs.com" ...
```

The value provided via `--field-val` overwrites the value provided in the
JSON fields file.

To explicitly unset a field value from CLI, use `--field-unset` (alias: `-fvu` - field
value unset), for example:

```shell
sipexer --field-unset rport --field-unset viabranch ...
```

If both `--field-val` and `--field-unset` are provided for the same field, the
unset operation takes precedence.

When sending out, before the template is evaluated, the following fields are also
added internally and will replace the corresponding `{{.name}}` (e.g., `{{.proto}}`)
in the template:

  * `afver` - address family version (`4` or `6`)
  * `localaddr` - local address - `ip:port`
  * `localip` - local ip
  * `localport` - local port
  * `proto` - lower(`proto`) (e.g., `udp`, `tcp`, ...)
  * `protoup` - upper(`proto`) (e.g., `UDP`, `TCP`, ...)
  * `sdpaf` - SDP address family (`IP4` or `IP6`)
  * `targetaddr` - remote address - `ip:port`
  * `targetip` - remote ip
  * `targetport` - remote port
  * `cr` - `\r`
  * `lf` - `\n`
  * `tab` - `\t`


## Alternatives

There are several alternatives that might be useful to consider:

  * `sipp` - SIP testing tool using XML-based scenarios
  * `sipsak` - SIP swiss army knife - SIP cli testing tool
  * `wsctl` - WebSocket cli tool with basic support for SIP
  * `baresip` - cli SIP softphone
  * `pjsua` - cli SIP softphone

## License

`GPLv3`

Copyright: `Daniel-Constantin Mierla` ([Asipto](https://www.asipto.com))

## Contributions

Contributions are welcome!

Fork and do pull requests:

  * https://github.com/miconda/sipexer
