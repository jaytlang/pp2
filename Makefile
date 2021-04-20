GOFLAGS = -race
TARGET = pp2

.PHONY: all clean
all: $(TARGET)

$(TARGET):
	go build $(GOFLAGS) -o $@

clean:
	$(RM) $(TARGET)

