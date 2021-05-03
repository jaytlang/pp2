package fs

/*
// Can't be called with dir == "/"
func (p *PigeonFs) walkToDir(dir string) (*inode.DirEnt, error) {
	dir = path.Clean(dir)
	els := strings.Split(dir, "/")
	d := &inode.DirEnt{
		Filename: "/",

		Inodenum: inode.RootInum,
	}

	for _, el := range els {
		i := inode.Geti(uint(d.Inodenum))
		if i.Mode != inode.Dir {
			return nil, errors.New("target is not a directory")
		}
		for _, a := range i.Addrs {
			b := bio.Bget(a)
			de := inode.DDecode(b.Data)
			if de.Filename == el {

			}
		}

	}
}

*/
