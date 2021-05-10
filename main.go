package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"pp2/bio"
	"pp2/fs"
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
	f := fs.Mount()

	for {
		fmt.Print("> ")
		ri, _ := rdr.ReadString('\n')
		ri = strings.Replace(ri, "\n", "", -1)
		i := strings.Split(ri, " ")

		switch i[0] {
		case "open":
			if len(i) != 2 {
				goto badcmd
			}
			res := f.Open(i[1])
			fmt.Printf("Opened file %s -> fd %d\n", i[1], res)
			continue

		case "read":
			if len(i) != 3 {
				goto badcmd
			}

			fd, err := strconv.ParseInt(i[1], 10, 64)
			if err != nil {
				goto badcmd
			}

			cnt, err := strconv.ParseUint(i[2], 10, 64)
			if err != nil {
				goto badcmd
			}

			res, err := f.Read(int(fd), uint(cnt))
			if err != nil {
				fmt.Printf("Read error: %s\n", err)
			} else {
				fmt.Printf("Read data: %s\n", res)
			}
			continue

		case "write":
			if len(i) != 3 {
				goto badcmd
			}

			fd, err := strconv.ParseInt(i[1], 10, 64)
			if err != nil {
				goto badcmd
			}

			res, err := f.Write(int(fd), i[2])
			if err != nil {
				fmt.Printf("Write error: %s\n", err)
			} else {
				fmt.Printf("Wrote %d bytes of data", res)
			}
			continue

		case "close":
			if len(i) != 2 {
				goto badcmd
			}

			fd, err := strconv.ParseInt(i[1], 10, 64)
			if err != nil {
				goto badcmd
			}

			f.Close(int(fd))
			fmt.Printf("Closed fd %d\n", fd)
			continue

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

		case "alloci":
			if len(i) != 1 {
				goto badcmd
			}
			if !inTxn {
				fmt.Printf("Not in transaction\n")
				continue
			}

			i := inode.Alloci(t, inode.File)
			i.Relse()
			fmt.Printf("Got inode %d\n", i.Serialnum)

		case "freei":
			if len(i) != 2 {
				goto badcmd
			}
			if !inTxn {
				fmt.Printf("Not in transaction\n")
				continue
			}

			inum, err := strconv.ParseUint(i[1], 10, 16)
			if err != nil {
				goto badcmd
			}

			i := inode.Geti(uint16(inum))
			err = i.Free(t)
			if err != nil {
				fmt.Printf("Error: %s\n", err.Error())
			} else {
				fmt.Printf("Freed %d\n", inum)
			}

		case "readi":
			if len(i) != 4 {
				goto badcmd
			}

			inum, err := strconv.ParseUint(i[1], 10, 16)
			if err != nil {
				goto badcmd
			}
			offset, err := strconv.ParseUint(i[2], 10, 64)
			if err != nil {
				goto badcmd
			}
			count, err := strconv.ParseUint(i[3], 10, 64)
			if err != nil {
				goto badcmd
			}

			res := inode.Readi(uint16(inum), uint(offset), uint(count))
			fmt.Printf("Read: %s\n", res)
			continue

		case "writei":
			if len(i) != 4 {
				goto badcmd
			}
			if !inTxn {
				fmt.Printf("not in transaction\n")
				continue
			}

			inum, err := strconv.ParseUint(i[1], 10, 16)
			if err != nil {
				goto badcmd
			}
			offset, err := strconv.ParseUint(i[2], 10, 64)
			if err != nil {
				goto badcmd
			}

			res, err := inode.Writei(t, uint16(inum), uint(offset), i[3])
			if err != nil {
				fmt.Printf("Err: %s\n", err.Error())
			} else {
				fmt.Printf("Wrote %d bytes\n", res)
			}
			continue

			/*
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

					res, _ := balloc.AllocBlocks(t, 1)
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

					balloc.RelseBlocks(t, []uint{uint(nr)})
					fmt.Printf("block freed\n")
			*/
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
