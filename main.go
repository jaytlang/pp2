package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"pp2/kvraft"
	"pp2/netdrv"
	"pp2/raft"
	"strings"
)

func runCli(c *kvraft.Clerk) {
	rdr := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		ri, _ := rdr.ReadString('\n')
		ri = strings.Replace(ri, "\n", "", -1)
		i := strings.Split(ri, " ")

		switch i[0] {
		case "get":
			if len(i) != 2 {
				goto badcmd
			}
			fmt.Printf("%s -> %s\n", i[1], c.Get(i[1]))
		case "put":
			if len(i) != 3 {
				goto badcmd
			}
			c.Put(i[1], i[2])
			fmt.Printf("%s now -> %s\n", i[1], i[2])
		case "append":
			if len(i) != 3 {
				goto badcmd
			}
			c.Append(i[1], i[2])
			fmt.Printf("%s += %s\n", i[1], i[2])
		}
		continue

	badcmd:
		fmt.Printf("Invalid arguments!\n")
	}
}

func printUsageMsgAndDie(err string) {
	fmt.Printf("Usage: ./pp2 <client | server>\n")
	fmt.Printf("Error: %s\n", err)
	os.Exit(1)
}

func main() {
	a := os.Args
	if len(a) != 2 {
		printUsageMsgAndDie("invalid number of arguments")
	} else if a[1] != "client" && a[1] != "server" {
		printUsageMsgAndDie("invalid second argument")
	}

	// Set up raft netconf and kv netconf
	rc := netdrv.MkDefaultNetConfig(true)
	kv := netdrv.MkDefaultNetConfig(false)

	if a[1] == "client" {
		c := kvraft.MakeClerk(kv)
		runCli(c)
	} else {
		me, err := rc.GetMe()
		if err != nil {
			log.Fatal(err)
		}
		kvraft.StartKVServer(rc, me, raft.MakePersister(), 4096)

		rdr := bufio.NewReader(os.Stdin)
		fmt.Printf("Press enter to kill kv...")
		rdr.ReadString('\n')
	}
}
