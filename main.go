package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"pp2/balloc"
	"pp2/bio"
	"pp2/inode"
	"pp2/jrnl"
	"pp2/kvraft"
	"pp2/netdrv"
	"pp2/raft"
	"strconv"
	"strings"
)

func runCli() {
	rdr := bufio.NewReader(os.Stdin)
	inTxn := false
	var t *jrnl.TxnHandle

	for {
		fmt.Print("> ")
		ri, _ := rdr.ReadString('\n')
		ri = strings.Replace(ri, "\n", "", -1)
		i := strings.Split(ri, " ")

		switch i[0] {
		case "begin":
			if len(i) != 1 {
				goto badcmd
			} else if inTxn {
				fmt.Printf("already in transaction\n")
				continue
			}
			t = jrnl.BeginTransaction()
			inTxn = true
			fmt.Printf("started a new transaction\n")

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

		case "write":
			if len(i) != 3 {
				goto badcmd
			} else if !inTxn {
				fmt.Printf("not in transaction\n")
				continue
			}

			nr, err := strconv.ParseUint(i[1], 10, 64)
			if err != nil {
				goto badcmd
			}

			blk := &bio.Block{
				Nr:   uint(nr),
				Data: i[2],
			}

			err = t.WriteBlock(blk)
			if err != nil {
				fmt.Printf("write failed: %s\n", err.Error())
			} else {
				fmt.Printf("wrote block successfully to log\n")
			}

		case "end":
			if !inTxn {
				fmt.Printf("not in transaction\n")
				continue
			}
			t.EndTransaction(false)
			t = nil
			inTxn = false
			fmt.Printf("transaction ended\n")

		case "abort":
			if !inTxn {
				fmt.Printf("not in transaction\n")
				continue
			}
			t.AbortTransaction()
			t = nil
			inTxn = false
			fmt.Printf("transaction ended\n")

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
		case "renew":
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

			berr := blk.Brenew()
			switch berr {
			case bio.OK:
				fmt.Printf("lock on block %s renewed\n", i[1])
			case bio.ErrNoLock:
				fmt.Printf("lock lease expired")
			}

		case "balloc":
			if !inTxn {
				fmt.Printf("not in transaction\n")
				continue
			}
			if len(i) != 1 {
				goto badcmd
			}

			res := balloc.AllocBlock(t)
			fmt.Printf("Got block %d\n", res)

		case "brelse":
			if len(i) != 2 {
				goto badcmd
			}
			if !inTxn {
				fmt.Printf("not in transaction\n")
				continue
			}

			nr, err := strconv.ParseUint(i[1], 10, 64)
			if err != nil {
				goto badcmd
			}

			balloc.RelseBlock(t, uint(nr))
			fmt.Printf("block freed\n")
		}
		continue

	badcmd:
		fmt.Printf("Invalid arguments!\n")
	}
}

func printUsageMsgAndDie(err string) {
	fmt.Printf("Usage: ./pp2 <client | server | ns> <nsAddr (localhost if args[1] == 'ns')>\n")
	fmt.Printf("Error: %s\n", err)
	os.Exit(1)
}

func main() {
	a := os.Args
	if len(a) != 3 {
		printUsageMsgAndDie("invalid number of arguments")
	} else if a[1] != "client" && a[1] != "server" && a[1] != "ns" {
		printUsageMsgAndDie("invalid second argument")
	}

	if a[1] == "ns" {
		netdrv.RunNameserver()
	} else if a[1] == "client" {
		bio.Binit(a[2])
		jrnl.InitSb()
		inode.InodeInit()
		runCli()

	} else {
		rc := netdrv.MkDefaultNetConfig(true, true, a[2])

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
