package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"pp2/bio"
	"pp2/kvraft"
	"pp2/netdrv"
	"pp2/raft"
	"strconv"
	"strings"
)

func runCli() {
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

			nr, err := strconv.ParseUint(i[1], 10, 64)
			if err != nil {
				goto badcmd
			}

			blk := bio.Bget(uint(nr))
			fmt.Printf("%s -> %s\n", i[1], blk.Data)

		case "push":
			if len(i) != 3 {
				goto badcmd
			}

			nr, err := strconv.ParseUint(i[1], 10, 64)
			if err != nil {
				goto badcmd
			}

			blk := &bio.Block{
				Nr:   uint(nr),
				Data: i[2],
			}

			berr := blk.Bpush()
			switch berr {
			case bio.OK:
				fmt.Printf("%s now -> %s\n", i[1], i[2])
			case bio.ErrBadSize:
				fmt.Printf("attempted to push too much data (blksz 4096 bytes!)\n")
			case bio.ErrNoLock:
				fmt.Printf("lock lease expired\n")
			}
		case "relse":
			if len(i) != 2 {
				goto badcmd
			}

			nr, err := strconv.ParseUint(i[1], 10, 64)
			if err != nil {
				goto badcmd
			}

			blk := &bio.Block{
				Nr: uint(nr),
			}

			berr := blk.Brelse()
			switch berr {
			case bio.OK:
				fmt.Printf("block %s released\n", i[1])
			case bio.ErrNoLock:
				fmt.Printf("lock lease expired\n")
			}
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

	if a[1] == "client" {
		bio.Binit()
		runCli()
	} else {
		rc := netdrv.MkDefaultNetConfig(true)

		me, err := rc.GetMe()
		if err != nil {
			log.Fatal(err)
		}

		kvraft.StartKVServer(rc, me, raft.MakePersister(), 50)

		rdr := bufio.NewReader(os.Stdin)
		fmt.Printf("Press enter to kill kv...")
		rdr.ReadString('\n')
	}
}
