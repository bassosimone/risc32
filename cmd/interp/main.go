package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/bassosimone/risc32/pkg/asm"
	"github.com/bassosimone/risc32/pkg/vm"
)

func main() {
	log.SetFlags(0)
	debug := flag.Bool("d", false, "enable debugging")
	filename := flag.String("f", "", "file to run")
	tty := flag.Bool("tty", false, "enable tty")
	verbose := flag.Bool("v", false, "be verbose")
	flag.Parse()
	if *filename == "" {
		log.Fatal("usage: interp [-d] [-tty] [-v] -f <assembly-code-file>")
	}
	machine := new(vm.VM)
	fp, err := os.Open(*filename)
	if err != nil {
		log.Fatal(err)
	}
	if *tty {
		stty, err := vm.TTYAcceptConn()
		if err != nil {
			log.Fatal(err)
		}
		defer stty.Close()
		machine.TTY = stty
	}
	defer fp.Close()
	var addr uint32
	for instr := range asm.StartAssembler(fp) {
		if instr.Error != nil {
			log.Fatal(instr.Error)
		}
		machine.M[addr] = instr.Instruction
		addr++
	}
	for {
		ci, err := machine.Fetch()
		if err != nil {
			log.Fatal(err)
		}
		if *verbose || (machine.StatusDebug()&vm.StatusDebugTracing) != 0 {
			log.Printf("vm: %s", machine)
			log.Printf("vm: %#032b %s\n", ci, vm.Disassemble(ci))
			log.Printf("vm: S[3]: %d", machine.S[3])
			log.Printf("vm: stack (r29): %d", machine.GPR[29])
		}
		if *debug || (machine.StatusDebug()&vm.StatusDebugStepping) != 0 {
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
