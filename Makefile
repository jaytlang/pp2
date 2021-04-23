GOFLAGS = -race
FMTFLAGS = -s -w
TARGET = pp2
TMPFILES = persiststate

.PHONY: all format clean
all: format
	go build $(GOFLAGS) -o $(TARGET)

format:
	gofmt $(FMTFLAGS) .

clean:
	$(RM) $(TARGET) $(TMPFILES)

