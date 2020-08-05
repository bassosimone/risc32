package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/bassosimone/risc32/pkg/vm"
)

func main() {
	log.SetFlags(0)
	debug := flag.Bool("d", false, "enable debugging")
	filename := flag.String("f", "", "file to run")
	verbose := flag.Bool("v", false, "be verbose")
	flag.Parse()
	if *filename == "" {
		log.Fatal("usage: vm [-d] [-v] -f <machine-code-file>")
	}
	fp, err := os.Open(*filename)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()
	machine, err := vm.LoadBytecode(fp)
	if err != nil {
		log.Fatal(err)
	}
	for {
		ci, err := machine.Fetch()
		if err != nil {
			log.Fatal(err)
		}
		if *verbose {
			log.Printf("vm: %s\n", machine)
			log.Printf("vm: %#032b %s\n", ci, vm.Disassemble(ci))
		}
		if *debug {
			log.Printf("vm: paused...")
			fmt.Scanln()
		}
		if err := machine.Execute(ci); err != nil {
			if errors.Is(err, vm.ErrHalted) {
				break
			}
			log.Fatal(err)
		}
	}
}
