GOFLAGS = -race
TARGET = pp2
TMPFILES = persiststate

.PHONY: all clean
all: $(TARGET)

$(TARGET):
	go build $(GOFLAGS) -o $@

clean:
	$(RM) $(TARGET) $(TMPFILES)

