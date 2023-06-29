# Compression
## Pure GO Huffman implementation

### Usage
```bash
// For Encode
./compress -C -i input.txt output.bin

// For Decode
./compress -D -i input.bin output.txt
```

Stats
```
with only same char
100mb ->  10kb

fully random char
100mb -> 93mb

executable binary
6mb -> 5.4mb
```
