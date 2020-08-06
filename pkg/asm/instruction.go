package asm

import (
	"fmt"
	"strconv"
)

// TODO(bassosimone): maybe create package pkg/spec where we can
// store the constants defining the ISA?

// The following constants define the opcodes. We have 5 bits to define
// opcodes, so up to 32 opcodes. While the opcodes here are related to
// the ones of RiSC-16, here we have more opcodes and also their values
// aren't necessarily aligned with the RiSC-16 architecture ones.
const (
	OpcodeHALT = uint32(iota) // auto-halt when hitting uninit mem
	OpcodeADD
	OpcodeADDI
	OpcodeNAND
	OpcodeLUI
	OpcodeSW
	OpcodeLW
	OpcodeBEQ
	OpcodeJALR
	OpcodeWSR
	OpcodeRSR
)

// Instruction is a parsed instruction.
type Instruction interface {
	// Err returns the error occurred processing the instruction. If this
	// function returns nil, then the instruction is valid.
	Err() error

	// Label returns the label associated with the instruction. If this
	// function returns nil, then there is no label.
	Label() *string

	// Line returns the line where the instruction appears in the input file.
	Line() int

	// Encode encodes the instruction. The table passed in input maps each
	// label to the corresponding offset in memory.
	Encode(labels map[string]int64, pc uint32) (uint32, error)
}

// InstructionErr is an error
type InstructionErr struct {
	Error  error
	Lineno int
}

// Err implements Instruction.Err
func (ia InstructionErr) Err() error {
	return ia.Error
}

// Label implements Instruction.Label
func (ia InstructionErr) Label() *string {
	return nil
}

// Line implements Instruction.Line
func (ia InstructionErr) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionErr) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	return 0, fmt.Errorf("%w because this is an error", ErrCannotEncode)
}

// NewParseError constructs a new parsed instruction
// that actually wraps a parsing error.
func NewParseError(err error) []Instruction {
	return []Instruction{InstructionErr{Error: err}}
}

var _ Instruction = InstructionErr{}

// InstructionADD is the ADD instruction
type InstructionADD struct {
	Lineno     int
	MaybeLabel *string
	RA         uint32
	RB         uint32
	RC         uint32
}

// Err implements Instruction.Err
func (ia InstructionADD) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionADD) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionADD) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionADD) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	var out uint32
	out |= (OpcodeADD & 0b1_1111) << 27
	out |= (ia.RA & 0b1_1111) << 22
	out |= (ia.RB & 0b1_1111) << 17
	out |= ia.RC & 0b1_1111
	return out, nil
}

var _ Instruction = InstructionADD{}

// InstructionADDI is the ADDI instruction
type InstructionADDI struct {
	Lineno     int
	MaybeLabel *string
	RA         uint32
	RB         uint32
	Imm        string
}

// Err implements Instruction.Err
func (ia InstructionADDI) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionADDI) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionADDI) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionADDI) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	var out uint32
	out |= (OpcodeADDI & 0b1_1111) << 27
	out |= (ia.RA & 0b1_1111) << 22
	out |= (ia.RB & 0b1_1111) << 17
	imm, err := ResolveImmediate(labels, ia.Imm, 17, ia.Lineno)
	if err != nil {
		return 0, err
	}
	out |= imm & 0b1_1111_1111_1111_1111
	return out, nil
}

var _ Instruction = InstructionADDI{}

// InstructionNAND is the NAND instruction
type InstructionNAND struct {
	Lineno     int
	MaybeLabel *string
	RA         uint32
	RB         uint32
	RC         uint32
}

// Err implements Instruction.Err
func (ia InstructionNAND) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionNAND) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionNAND) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionNAND) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	var out uint32
	out |= (OpcodeNAND & 0b1_1111) << 27
	out |= (ia.RA & 0b1_1111) << 22
	out |= (ia.RB & 0b1_1111) << 17
	out |= ia.RC & 0b1_1111
	return out, nil
}

var _ Instruction = InstructionNAND{}

// InstructionLUI is the LUI instruction
type InstructionLUI struct {
	Lineno     int
	MaybeLabel *string
	RA         uint32
	Imm        string
}

// Err implements Instruction.Err
func (ia InstructionLUI) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionLUI) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionLUI) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionLUI) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	var out uint32
	out |= (OpcodeLUI & 0b1_1111) << 27
	out |= (ia.RA & 0b1_1111) << 22
	imm, err := ResolveImmediate(labels, ia.Imm, 32, ia.Lineno)
	if err != nil {
		return 0, err
	}
	out |= (imm >> 10)
	return out, nil
}

var _ Instruction = InstructionLUI{}

// InstructionSW is the SW instruction
type InstructionSW struct {
	Lineno     int
	MaybeLabel *string
	RA         uint32
	RB         uint32
	Imm        string
}

// Err implements Instruction.Err
func (ia InstructionSW) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionSW) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionSW) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionSW) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	var out uint32
	out |= (OpcodeSW & 0b1_1111) << 27
	out |= (ia.RA & 0b1_1111) << 22
	out |= (ia.RB & 0b1_1111) << 17
	imm, err := ResolveImmediate(labels, ia.Imm, 17, ia.Lineno)
	if err != nil {
		return 0, err
	}
	out |= imm & 0b1_1111_1111_1111_1111
	return out, nil
}

var _ Instruction = InstructionSW{}

// InstructionLW is the LW instruction
type InstructionLW struct {
	Lineno     int
	MaybeLabel *string
	RA         uint32
	RB         uint32
	Imm        string
}

// Err implements Instruction.Err
func (ia InstructionLW) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionLW) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionLW) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionLW) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	var out uint32
	out |= (OpcodeLW & 0b1_1111) << 27
	out |= (ia.RA & 0b1_1111) << 22
	out |= (ia.RB & 0b1_1111) << 17
	imm, err := ResolveImmediate(labels, ia.Imm, 17, ia.Lineno)
	if err != nil {
		return 0, err
	}
	out |= imm & 0b1_1111_1111_1111_1111
	return out, nil
}

var _ Instruction = InstructionLW{}

// InstructionBEQ is the BEQ instruction
type InstructionBEQ struct {
	Lineno     int
	MaybeLabel *string
	RA         uint32
	RB         uint32
	Imm        string
}

// Err implements Instruction.Err
func (ia InstructionBEQ) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionBEQ) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionBEQ) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionBEQ) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	var out uint32
	out |= (OpcodeBEQ & 0b1_1111) << 27
	out |= (ia.RA & 0b1_1111) << 22
	out |= (ia.RB & 0b1_1111) << 17
	imm, err := ResolveImmediate(labels, ia.Imm, 32, ia.Lineno)
	if err != nil {
		return 0, err
	}
	var target int64 = int64(imm) - int64(pc) - 1
	offset, err := CastToUint32(target, 17, ia.Lineno)
	if err != nil {
		return 0, err
	}
	out |= offset & 0b1_1111_1111_1111_1111
	return out, nil
}

var _ Instruction = InstructionBEQ{}

// InstructionJALR is the JALR instruction
type InstructionJALR struct {
	Lineno     int
	MaybeLabel *string
	RA         uint32
	RB         uint32
}

// Err implements Instruction.Err
func (ia InstructionJALR) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionJALR) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionJALR) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionJALR) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	var out uint32
	out |= (OpcodeJALR & 0b1_1111) << 27
	out |= (ia.RA & 0b1_1111) << 22
	out |= (ia.RB & 0b1_1111) << 17
	return out, nil
}

var _ Instruction = InstructionJALR{}

// InstructionHALT is the HALT instruction
type InstructionHALT struct {
	Lineno     int
	MaybeLabel *string
}

// Err implements Instruction.Err
func (ia InstructionHALT) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionHALT) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionHALT) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionHALT) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	var out uint32
	out |= (OpcodeHALT & 0b1_1111) << 27
	return out, nil
}

var _ Instruction = InstructionHALT{}

// InstructionLLI is the LLI pseudo-instruction
type InstructionLLI struct {
	Lineno     int
	MaybeLabel *string
	RA         uint32
	Imm        string
}

// Err implements Instruction.Err
func (ia InstructionLLI) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionLLI) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionLLI) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionLLI) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	var out uint32
	out |= (OpcodeADDI & 0b1_1111) << 27
	out |= (ia.RA & 0b1_1111) << 22
	out |= (ia.RA & 0b1_1111) << 17
	imm, err := ResolveImmediate(labels, ia.Imm, 32, ia.Lineno)
	if err != nil {
		return 0, err
	}
	out |= (imm & 0b11_1111_1111)
	return out, nil
}

var _ Instruction = InstructionLLI{}

// InstructionDATA is the .SPACE or .FILL pseudo-instruction
type InstructionDATA struct {
	Lineno     int
	MaybeLabel *string
	Value      uint32
}

// Err implements Instruction.Err
func (ia InstructionDATA) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionDATA) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionDATA) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionDATA) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	return ia.Value, nil
}

var _ Instruction = InstructionDATA{}

// InstructionWSR is the WSR instruction
type InstructionWSR struct {
	Lineno     int
	MaybeLabel *string
	RA         uint32
	Imm        string
}

// Err implements Instruction.Err
func (ia InstructionWSR) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionWSR) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionWSR) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionWSR) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	var out uint32
	out |= (OpcodeWSR & 0b1_1111) << 27
	out |= (ia.RA & 0b1_1111) << 22
	imm, err := ResolveImmediate(labels, ia.Imm, 32, ia.Lineno)
	if err != nil {
		return 0, err
	}
	out |= (imm >> 10)
	return out, nil
}

var _ Instruction = InstructionWSR{}

// InstructionRSR is the RSR instruction
type InstructionRSR struct {
	Lineno     int
	MaybeLabel *string
	RA         uint32
	Imm        string
}

// Err implements Instruction.Err
func (ia InstructionRSR) Err() error {
	return nil
}

// Label implements Instruction.Label
func (ia InstructionRSR) Label() *string {
	return ia.MaybeLabel
}

// Line implements Instruction.Line
func (ia InstructionRSR) Line() int {
	return ia.Lineno
}

// Encode implements Instruction.Encode
func (ia InstructionRSR) Encode(labels map[string]int64, pc uint32) (uint32, error) {
	var out uint32
	out |= (OpcodeRSR & 0b1_1111) << 27
	out |= (ia.RA & 0b1_1111) << 22
	imm, err := ResolveImmediate(labels, ia.Imm, 32, ia.Lineno)
	if err != nil {
		return 0, err
	}
	out |= (imm >> 10)
	return out, nil
}

var _ Instruction = InstructionRSR{}

// ResolveImmediate resolves the value of an immediate
func ResolveImmediate(
	labels map[string]int64, name string, bits, lineno int) (uint32, error) {
	value, err := strconv.ParseInt(name, 0, 64)
	if err != nil {
		var found bool
		value, found = labels[name]
		if !found {
			return 0, fmt.Errorf("%w because label '%s' is missing", ErrCannotEncode, name)
		}
		// fallthrough
	}
	return CastToUint32(value, bits, lineno)
}

// CastToUint32 casts the given value to uint32
func CastToUint32(value int64, bits, lineno int) (uint32, error) {
	if bits < 1 || bits > 32 {
		panic("bits value out of range")
	}
	if value < -(1<<(bits-1)) || value > ((1<<(bits-1))-1) {
		return 0, fmt.Errorf("%w for %d-bit range on line %d", ErrOutOfRange, bits, lineno)
	}
	return uint32(value), nil
}
