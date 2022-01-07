package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
)

const (
	signatureCap = 8

	endChunkType = "IEND"
)

var (
	signature = [signatureCap]byte{137, 80, 78, 71, 13, 10, 26, 10}
)

func cmpBytes(exp, got []byte) bool {
	for i, b := range exp {
		if b != got[i] {
			return false
		}
	}

	return true
}

func readBytesOrPanic(f *os.File, buf []byte) {
	_, err := f.Read(buf)
	if err != nil {
		panic(fmt.Sprintf("falied to read from file %s: %s", f.Name(), err.Error()))
	}
}

func writeBytesOrPanic(f *os.File, b []byte) {
	_, err := f.Write(b)
	if err != nil {
		panic(fmt.Sprintf("falied to write to file %s: %s", f.Name(), err.Error()))
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "USAGE: pngp -i <input.png> [-o <output.png>] [-s <secret to put>]\n")
	os.Exit(1)
}

func main() {
	inputFileName := flag.String("i", "", "Input file name")
	outputFileName := flag.String("o", "output.png", "Output file name")
	secretWord := flag.String("s", "", "Output file name")

	flag.Parse()

	if *inputFileName == "" || *outputFileName == "" {
		usage()
	}

	inputFile, err := os.Open(*inputFileName)
	if err != nil {
		panic(fmt.Sprintf("failed to open input file %s: %s\n", *inputFileName, err.Error()))
	}

	outputFile, err := os.Create(*outputFileName)
	if err != nil {
		panic(fmt.Sprintf("failed to open output file %s: %s\n", *inputFileName, err.Error()))
	}

	defer func() {
		inputFile.Close()
		outputFile.Close()
	}()

	sig := make([]byte, signatureCap)
	inputFile.Read(sig)

	if !cmpBytes(signature[:], sig) {
		panic(fmt.Sprintf("ERROR: invalid PNG signature %v\n", sig))
	}

	outputFile.Write(sig)

	var totalSize int

	for {
		chunkSizeBuf := make([]byte, 4)

		readBytesOrPanic(inputFile, chunkSizeBuf)
		writeBytesOrPanic(outputFile, chunkSizeBuf)

		chunkSize := binary.BigEndian.Uint32(chunkSizeBuf)
		totalSize += int(chunkSize)

		chunkTypeBuf := make([]byte, 4)
		readBytesOrPanic(inputFile, chunkTypeBuf)
		writeBytesOrPanic(outputFile, chunkTypeBuf)

		chunkBuf := make([]byte, chunkSize)
		readBytesOrPanic(inputFile, chunkBuf)
		writeBytesOrPanic(outputFile, chunkBuf)

		crcBuf := make([]byte, 4)
		readBytesOrPanic(inputFile, crcBuf)
		writeBytesOrPanic(outputFile, crcBuf)

		// insert chunk
		if string(chunkTypeBuf) == "IHDR" { // Type of the first chunk
			secretSizeBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(secretSizeBuf, uint32(len(*secretWord)+1)) // + 1 extra symbol for separator

			writeBytesOrPanic(outputFile, secretSizeBuf)
			writeBytesOrPanic(outputFile, []byte("jOJO"))
			writeBytesOrPanic(outputFile, []byte("_"+*secretWord))
			writeBytesOrPanic(outputFile, []byte{0, 0, 0, 0})
		}

		fmt.Println("Chunk size: ", chunkSize)
		fmt.Println("Chunk type: ", string(chunkTypeBuf))
		fmt.Printf("CRC: 0x%s", hex.EncodeToString(crcBuf))
		fmt.Println("\n=================================")

		if string(chunkTypeBuf) == endChunkType {
			break
		}
	}

	fmt.Println("Total size int bits: ", totalSize)
}
