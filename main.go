package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
)

type HuffmanNode struct {
	Value      byte
	Frequency  int
	LeftChild  *HuffmanNode
	RightChild *HuffmanNode
}

type FileHeader struct {
	TreeSize       int64
	DataSize       int64
	CompressAmount int8
}

type HuffmanRoot struct {
	root *HuffmanNode
}

func StringToByteArray(input string) []byte {
	var byteArray []byte = make([]byte, len(input)/8)
	currentByte := byte(0)
	bitCount := 0
	bit := 0

	for i, char := range input {
		bit, _ = strconv.Atoi(string(char))

		currentByte = (currentByte << 1) | byte(bit)
		bitCount++

		if bitCount == 8 {
			byteArray[i/8] = currentByte
			currentByte = byte(0)
			bitCount = 0
		}
	}

	if bitCount > 0 {
		currentByte <<= (8 - bitCount)
		byteArray = append(byteArray, currentByte)
	}

	return byteArray
}

func readChunks(filename string, chunkSize int) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()

	// Calculate the number of chunks
	numChunks := int(fileSize) / chunkSize
	if int(fileSize)%chunkSize != 0 {
		numChunks++
	}

	// Create a byte array to store the file content
	fileContent := make([]byte, 0, fileSize)

	// Read file in chunks
	buffer := make([]byte, chunkSize)
	for i := 0; i < numChunks; i++ {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		fileContent = append(fileContent, buffer[:n]...)
	}

	return fileContent, nil
}

func (encoder *HuffmanRoot) Encode(data []byte) ([]byte, error) {
	codes := generateHuffmanCodes(encoder.root)
	compressed := compressData(data, codes)
	return compressed, nil
}

func compressData(data []byte, codes map[byte]string) []byte {
	var buffer bytes.Buffer
	for _, b := range data {
		_, err := buffer.WriteString(codes[b])
		if err != nil {
			fmt.Println("error")
		}
	}

	ham := StringToByteArray(buffer.String())
	return ham
}

func (encoder *HuffmanRoot) SerializeTree() ([]byte, error) {
	buffer := new(bytes.Buffer)
	enc := gob.NewEncoder(buffer)
	err := enc.Encode(encoder.root)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (encoder *HuffmanRoot) DeserializeTree(data []byte) error {
	buffer := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buffer)
	err := dec.Decode(&encoder.root)
	return err
}

func (encoder *HuffmanRoot) BuildTree(data []byte) {
	frequencies := make(map[byte]int)
	for _, b := range data {
		frequencies[b]++
	}

	encoder.root = buildHuffmanTree(frequencies)
}

func buildHuffmanTree(frequencies map[byte]int) *HuffmanNode {
	var nodes []*HuffmanNode
	var left, right *HuffmanNode

	// Create leaf nodes for each symbol
	for symbol, freq := range frequencies {
		nodes = append(nodes, &HuffmanNode{
			Value:     symbol,
			Frequency: freq,
		})
	}

	for len(nodes) > 1 {
		// Sort nodes by frequency (ascending order)
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Frequency < nodes[j].Frequency
		})

		// Take two nodes with the lowest frequency
		left = nodes[0]
		right = nodes[1]

		// Create a parent node with the sum of frequencies
		parent := &HuffmanNode{
			Frequency:  left.Frequency + right.Frequency,
			LeftChild:  left,
			RightChild: right,
		}

		// Remove the two nodes and add the parent node
		nodes = nodes[2:]
		nodes = append(nodes, parent)
	}

	// Return the root node
	return nodes[0]
}

func generateHuffmanCodes(root *HuffmanNode) map[byte]string {
	codes := make(map[byte]string, 0)
	buildHuffmanCodes(root, "", codes)
	return codes
}

func buildHuffmanCodes(node *HuffmanNode, code string, codes map[byte]string) {
	// Recursive calls for left and right children
	if node.LeftChild != nil {
		buildHuffmanCodes(node.LeftChild, code+"0", codes)
	}
	if node.RightChild != nil {
		buildHuffmanCodes(node.RightChild, code+"1", codes)
	}

	if node.LeftChild == nil && node.RightChild == nil {
		codes[node.Value] = code
		return
	}
}

func EncodeLoop(data []byte, i *int8) ([]byte, error) {
	var binBuffer bytes.Buffer
	encoder := &HuffmanRoot{}
	encoder.BuildTree(data)
	compressed, err := encoder.Encode(data)
	if err != nil {
		return compressed, err
	}

	treeData, err := encoder.SerializeTree()
	if err != nil {
		return compressed, err
	}

	header := FileHeader{
		TreeSize:       int64(len(treeData)),
		DataSize:       int64(len(data)),
		CompressAmount: *i,
	}

	err = binary.Write(&binBuffer, binary.LittleEndian, header)
	if err != nil {
		return compressed, err
	}

	err = binary.Write(&binBuffer, binary.LittleEndian, treeData)
	if err != nil {
		return compressed, err
	}

	err = binary.Write(&binBuffer, binary.LittleEndian, compressed)
	if err != nil {
		return compressed, err
	}

	if len(binBuffer.Bytes()) < len(data) {
		*i++
		data, err = EncodeLoop(binBuffer.Bytes(), i)
		if err != nil {
			return data, err
		}
	} else {
		data = binBuffer.Bytes()
	}

	return data, err
}

func EncodeAndLargeFile(inputFilePath, outputFilePath string) error {
	data, err := readChunks(inputFilePath, 1024)
	if err != nil {
		return err
	}

	var i int8 = 0
	compressed, err := EncodeLoop(data, &i)
	if err != nil {
		return err
	}

	file, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(compressed)
	if err != nil {
		return err
	}

	return nil
}

func DecodeLargeFile(filePath, decodedFilePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var binBuffer bytes.Buffer
	_, err = binBuffer.ReadFrom(file)
	if err != nil {
		return err
	}

	for {
		var header FileHeader
		err = binary.Read(&binBuffer, binary.LittleEndian, &header)
		if err != nil {
			return err
		}

		treeData := make([]byte, header.TreeSize)
		err = binary.Read(&binBuffer, binary.LittleEndian, &treeData)
		if err != nil {
			return err
		}

		encoder := &HuffmanRoot{}
		err = encoder.DeserializeTree(treeData)
		if err != nil {
			return err
		}

		decoded := decodeData(binBuffer.Bytes(), encoder.root, header.DataSize)

		binBuffer.Reset()
		_, err = binBuffer.Write(decoded)
		if err != nil {
			return err
		}

		if header.CompressAmount == 0 {
			break
		}
	}

	err = os.WriteFile(decodedFilePath, binBuffer.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

func decodeData(data []byte, root *HuffmanNode, size int64) []byte {
	var decoded []byte
	node := root
	var j int64 = 0

	for _, b := range data {
		for i := 7; i >= 0; i-- {
			bit := (b >> i) & 1
			if bit == 0 {
				node = node.LeftChild
			} else {
				node = node.RightChild
			}

			if node.LeftChild == nil && node.RightChild == nil {
				j++
				decoded = append(decoded, node.Value)
				node = root
				// For not used bits
				if j == size {
					break
				}
			}
		}
	}
	return decoded
}

func main() {
	compress := flag.Bool("C", false, "-C Compress")
	decompress := flag.Bool("D", false, "-D Compress")
	inputFilePath := flag.String("i", "", "-i filename")
	outputFilePath := flag.String("o", "", "-i filename")

	flag.Parse()

	if *inputFilePath == "" || *outputFilePath == "" {
		fmt.Println("input or output cannot be empty")
		fmt.Println(*inputFilePath, *outputFilePath)
		return
	}

	if *compress {
		err := EncodeAndLargeFile(*inputFilePath, *outputFilePath)
		if err != nil {
			fmt.Printf("Error encoding and saving: %s\n", err.Error())
			return
		}
	}

	if *decompress {
		err := DecodeLargeFile(*inputFilePath, *outputFilePath)
		if err != nil {
			fmt.Printf("Error decoding and saving: %s\n", err.Error())
			return
		}
	}
}
