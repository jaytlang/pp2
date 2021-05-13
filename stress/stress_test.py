import sys

currentfd = 0
def exec(id, file):
    global currentfd

    print(f"open {file}")
    myfd = currentfd
    currentfd += 1
    print(f"read {myfd} 9999999999")
    print(f"write {myfd} {id}-{myfd}_")
    print(f"close {myfd}")

def run(id, files, n):
    for _ in range(n):
        for file in files:
            exec(id, file)

def mkfiles(n):
    files = []
    for i in range(n):
        files.append(f"test_{i}")
    return files

nfiles = 5
niters = 5
if __name__ == '__main__':
    assert len(sys.argv) == 2
    id = sys.argv[1]
    files = mkfiles(nfiles)
    run(id, files, niters)