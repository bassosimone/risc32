// Package asm contains the RiSC-32 assembler.
//
// See the documentation of the vm package for more information
// about the instruction set and the bytecode format.
package asm

import (
	"fmt"
	"io"
	"math"
)

// InstructionOrError contains either an assembled instruction
// or an error that occurred during the assemblation.
type InstructionOrError struct {
	Instruction uint32
	Error       error
	Lineno      int
}

// Encode encodes the current instruction or returns an error.
func (ioe InstructionOrError) Encode() (string, error) {
	if ioe.Error != nil {
		return "", ioe.Error
	}
	return fmt.Sprintf(
		"0x%08x\t# 0b%032b - line: %d\n", ioe.Instruction, ioe.Instruction, ioe.Lineno,
	), nil
}

// StartAssembler starts the assembler in a background goroutine an
// returns a sequence of InstructionOrError.
func StartAssembler(r io.Reader) <-chan InstructionOrError {
	out := make(chan InstructionOrError)
	go AssemblerAsync(r, out)
	return out
}

// AssemblerAsync runs the assembler. It reads from the input reader
// and it writes InstructionOrError on the output channel.
func AssemblerAsync(r io.Reader, out chan<- InstructionOrError) {
	defer close(out)
	var idx int64
	labels := make(map[string]int64)
	var instructions []Instruction
	for instr := range StartParsing(StartLexing(r)) {
		if instr.Err() != nil {
			out <- InstructionOrError{Error: instr.Err(), Lineno: instr.Line()}
			return
		}
		if instr.Label() != nil {
			labels[*instr.Label()] = idx
		}
		instructions = append(instructions, instr)
		idx++
	}
	for pc, instr := range instructions {
		if pc > math.MaxUint32 {
			out <- InstructionOrError{Error: ErrTooManyInstructions, Lineno: instr.Line()}
			return
		}
		encoded, err := instr.Encode(labels, uint32(pc))
		if err != nil {
			out <- InstructionOrError{Error: err, Lineno: instr.Line()}
			continue
		}
		out <- InstructionOrError{Instruction: encoded, Lineno: instr.Line()}
	}
}
