package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/bassosimone/risc32/pkg/asm"
)

func main() {
	log.SetFlags(0)
	filename := flag.String("f", "", "file to process")
	flag.Parse()
	if *filename == "" {
		log.Fatal("usage: asm -f <assembly-code-file>")
	}
	fp, err := os.Open(*filename)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()
	for instr := range asm.StartAssembler(fp) {
		out, err := instr.Encode()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(out)
	}
}
