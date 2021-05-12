# Integration testing procedure

## Setup
To test pp2, you'll need a Go toolchain and at least 4 machines.
One of the machines is a *nameserver*, which orchestrates Raft
server discovery and tells clients which IP addresses are associated
with which servers. You will need two or more *Raft servers*, which
constitute the distributed block device and manage replication of
files. Lastly, you will need at least one client.

## Launching PP2
pp2 accepts the following command line arguments:
```
./pp2 < server | client | ns > <IPv4 address>
```
Argument one indicates whether this machine is a Raft server,
a client, or the nameserver respectively. The second argument
indicates the IP address of the nameserver - it is recommended
you set this to `localhost` if you are invoking `./pp2 ns`.

You should start a nameserver first, followed by all Raft servers
(at which point the Raft servers will print out diagnostic info
to reflect a successful leader election), then clients. Note that
pp2 is currently hard-configured to support 3 servers, but this
can be changed easily. The number of clients is unlimited.

Note that running ./refresh.sh on all servers will reset the
filesystem to an empty state, which is required for the below
tests.

## Testing PP2
We have several unit tests written for the various layers of PP2.
These can be run using `go test ./...`. Our recommended partitions
to integration test PP2 are as follows:

_Read_:
* one read vs. multiple reads serially
* zeroed seek ptr. vs. non-zero seek pointer
* count = 0; count <= len(file); count > len(file)
* read on one client equal to read on other client after write

_Write_:
* zeroed seek ptr. vs. non-zero seek pointer
* one write vs. multiple writes serially
* overwriting, appending data
* read of written bytes by another client returns those bytes after commit
* other client's written bytes overwrite data written by that client

_Open_:
* open one file, open multiple files
* create vs. open existing file
* create reflects on different client
* open an already opened file and verify seek pointers are independent

_Close_:
* close one file, close multiple files
* use fd after close (=FAIL)




## Test 1: Basic Test
*Covers*:
* `read/oneread`
* `read/zeroedseekptr`
* `read/count>len(file)`
* `write/onewrite`
* `write/zeroedseekptr`
* `write/appending`
* `open/oneopen`
* `open/create`
* `open/existing`
* `close/oneclose`

Note: the notation `cn` reflects a command for client `n`, and the sigil notation `$v` reflects a variable we call `v` for convenience

*Procedure*: 
```
c0: open $f -> $fd1
c0: write $fd1 hello
c0: close $fd1
c0: open $f -> $fd2
c0: read $fd2 999
c0: close $fd
```

*Expected Behavior*:
No crashes/errors, the read should display 'hello'.

## Test 2: Multiple Reads/Write to Same File
*Covers*:
* `read/multiread`
* `read/nonzeroseek`
* `read/count<=len(file)`
* `write/multiwrite`
* `write/nonzeroseek`
* `open/multiopen`
* `close/multiclose`

*Procedure*: 
```
c0: open $f -> $fd1
c0: write $fd1 hello
c0: write $fd1 goodday
c0: close $fd1

c0: open $f -> $fd2
c0: write $fd2 goodbye
c0: write $fd2 friend
c0: close $fd2

c0: open $f -> $fd3
c0: read $fd3 4
c0: read $fd3 9
c0: close $fd2
```

*Expected Behavior*:
No crashes/errors. The first read should output 'good', and second read should output 'read byefriend'.

## Test 3: Basic Persistence

*Covers*:
* `read/count=0`
* `open/createreflects`

*Procedure*: 
```
c0: open $f -> $fd1
c1: open $g -> $fd2
c0: read $f 50
c1: read $g 50
c0: close $f
c1: close $g

c1: open $f -> $fd3
c0: open $g -> $fd4
c1: read $f 50
c0: read $g 50
c1: close $f
c0: close $g
```

*Expected Behavior*:
No crashes/errors. All reads should return the empty string. The first two opens should result in visible creates + commit to the log, while the second two should 'find' existing files and make no such commit.

## Test 4: Double Open, Double Close

*Covers*:
* `open/independentseekptr`
* `close/fdafterclose`

*Procedure*:
```
c0: open $f -> $fd1
c0: open $f -> $fd2
c0: write $fd1 yeeteom
c0: read $fd2 50

c0: close $fd1
c0: close $fd2
c0: close $fd1
```

*Expected Behavior*:
No crashes/errors. The read should output the same string written by the write call (yeeteom). All closes should appear to succeed.

## Test 5: POSIX Consistency

*Covers*
* `read/consistent`
* `write/consistent`
* `write/otheroverwrite`

*Procedure*:
```
c0: open $f -> $fd1
c0: write $fd1 kek
c0: close $fd1

c0: open $f -> $fd2
c0: read $fd2 5
c0: close $fd2

c1: open $f -> $fd3
c1: read $fd3 5
c1: close $fd3

c1: open $f -> $fd4
c1: write $fd4 lol
c1: close $fd4

c0: open $f -> $fd5
c0: read $fd5 5
c0: close $fd5
```

*Expected Behavior*:
No crashes/errors. Read 1 should output 'kek', read 2 should output 'kek', and read 3 should output 'lol'.
