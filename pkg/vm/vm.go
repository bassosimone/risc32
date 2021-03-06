// Package vm contains the RiSC-32 VM.
//
// The architecture of this VM is inspired to that of the RiSC-16
// architecture <https://user.eng.umd.edu/~blj/RiSC/>.
//
// Instruction format
//
// Each instruction is 32 bits wide. We have three instructions formats:
//
// 1. RRR (register, register, register);
// 2. RRI (register, register, immediate);
// 3. RI (register, immediate).
//
// The following is the RRR format:
//
//     <Opcode:5><RegisterA:5><RegisterB:5><Unused:12><RegisterC:5>
//
// The following is the RRI format:
//
//     <Opcode:5><RegisterA:5><RegisterB:5><SignedImmediate:17>
//
// The following is the RI format:
//
//     <Opcode:5><RegisterA:5><Immediate:22>
//
// Bytecode format
//
// Instructions are serialized as 32-bit unsigned numbers. One instruction per
// line with an optional comment after the number. For example:
//
//     0x00000000   # HALT - line 1234
//
// The comment, if any, will be discarded. The format of the output number
// MUST be hexadecimal with a leading 0x prefix. It does not necessarily need
// to have a bunch of leading zeroes, but that would be nice.
//
// Instruction set
//
// This VM implements all the instructions of the RiSC-16. Like in the RiSC-16,
// JALR is used for halting and traps. We additionally implement:
//
// WSR (Write Status Register - RI format): writes the content of the specified
// general purpose register RA to the status register indicated by the specified
// immediate. This operation fails if we are running in user mode.
//
// RSR (Read Status Register): like WSR except that it reads a status register.
//
// Status Registers
//
// The status registers can only be accessed using RSR and WSR. When the
// UserMode bit is set, accessing status registers causes a fault.
//
// The status register with index 0 contains the processor flags. It currently
// defines the following bit flags:
//
//     <Unused: 29><Flags: 3>
//
// The following flags are defined:
//
// - UserMode (1<<0): set when we're in user mode
// - Paging (1<<1): whether paging is ON
// - Interrupts (1<<2): whether interrupts are ON
// - DebugStepping (1<<3): turns on stepping
// - DebugTracing (1<<4): turns on tracing
//
// The status register with index 1 contains the address in memory of the
// page table. The page table contains 1,024 32-bit entries. We use the page
// table only when the Paging flag is set. The page table must be aligned
// to a 1<<10 boundary, otherwise the machine halts.
//
// The status register with index 2 contains the address in memory of the
// interrupt handlers vector. This table contains 16 32-bit entries. We only
// use this table when the Interrupts flag is set. Also the interrupt table
// must be aligned to a 1<<10 boundary, otherwise the machine halts.
//
// The status register with index 3 contains the address in memory of the
// stack that should be used by interrupts. This value must be 1<<10 aligned
// like the page table and the interrupt handlers vector.
//
// Attempting to access a non-existent status register causes a fault.
//
// Page table
//
// Each entry in the page table takes 32 bits. We have 1,024 entries in
// total inside of the page table. The kernel must allocate the page table
// in a specific place and make sure it is protected, if needed.
//
// When paging is enabled, addresses are virtual addresses as follows:
//
//     <PageID: 22><Address: 10>
//
// Status register 1 contains the address the page table. By adding the PageID
// offset to such address, we fetch the corresponding entry.
//
// The entry itself is as follows:
//
//     <BaseAddr: 22><Flags: 10>
//
// The BaseAddr contains the base address of the corresponding page. The flags
// apply the following restrictions to the page:
//
// - `X` (1<<0): true if the page contains executable code
// - `W` (1<<1): true if the page is writeable
// - `R` (1<<2): true if the page is readable
//
// When the code accesses a user page without the proper restrictions, the
// processor will emit a fault and possibly terminate.
//
// A zeroed entry in the page table always causes a fault.
//
// Interrupts
//
// We have 32-bit 16 handlers. Each handler is the address of the handler
// routine to jump to. The hardware saves the status register, the next
// program counter, and the stack pointer. Then, it clears UserMode, Interrupts,
// and Paging, and transfers the control to the specified routine.
//
// Because the interrupt service routine runs with interrupts disabled, you
// are not supposed to receive more interrupts until done. Because it runs with
// paging disabled, when you install the interrupt service routine, you must
// make sure that you install an absolute memory address. This is done to ensure
// you must jump to the service routine even if paging is such that you'd not
// otherwise be able to jump to the routine address.
//
// The interrupt ID is indicated by the immediate and it is used to choose
// the proper handler in the table indicated by status register 2. We handle
// 16 interrupts. Any value of the interrupt not between 0 and 15 (inclusive)
// is mapped to zero. The default action of interrupt zero should be to stop
// the machine but some operations may be performed before that.
//
// The following IRQs are defined:
//
// - IrqHALT (0): asks the OS to halt
// - IrqClock (1): the clock needs attention
// - IrqTTY (2): the TTY needs attention
//
// The IRET instruction implements returning from the interrupt.
//
// Memory mapped I/O
//
// There is a bunch of memory locations reserved to memory mapped I/O (MMIO).
//
// Clock
//
// The clock uses the following MMIO locations:
//
// - MMClockFrequency (1<<17|0): this is the number of milliseconds after
// which you want the clock to generate an interrupt.
//
// TTY
//
// By default there is no attached TTY. If you attach a TTY before booting
// the machine and enable interrupts, you will need to service them.
//
// When there is an attached TTY, the following locations in memory will
// become useful for MMIO:
//
// - MMTTYStatus (1<<17|1): read the status of the TTY
// - MMTTYIn (1<<17|2): input from the TTY
// - MMTTYOut (1<<17|3): output for the TTY
//
// The MMTTYStatus word contains the status. The following bits matter:
//
// - TTYIn (1<<0): MMTTYIn contains a valid character
// - TTYOut (1<<1): MMTTYOut contains a valid character
//
// The MTTYIn word contains the next incoming char in the lowest byte of the
// word. A new incoming character causes an IrqTTY interrupt and the TTYIn bit
// will be set inside the MMTTYStatus word. The interrupt handler must read
// the character, dispatch it, and clear the TTYIn bit.
//
// The MTTYOut word contains the next outgoing char in the lowest byte of the
// word. The kernel should write into such word only if the TTYOut bit isn't
// set. Then it should set the bit so that the hardware delivers the char. When
// the delivery is complete, the hardware will clear TTYOut.
package vm

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"
)

// The following constants define the opcodes. We have 5 bits to define
// opcodes, so up to 32 opcodes. While the opcodes here are related to
// the ones of RiSC-16, here we have more opcodes and also their values
// aren't necessarily aligned with the RiSC-16 architecture ones.
const (
	// RiSC-16 like operations -- note that JALR is the first operation
	// so that zero initialized memory stops the VM when we are not using
	// interrupts, which is a quite handy feature.
	OpcodeJALR = uint32(iota)

	OpcodeADD
	OpcodeADDI
	OpcodeNAND
	OpcodeLUI
	OpcodeSW
	OpcodeLW
	OpcodeBEQ

	// Extended operations
	OpcodeWSR
	OpcodeRSR
	OpcodeIRET
)

const (
	// MemorySize is the memory size in 32-bit-wide words.
	MemorySize = 1 << 20

	// NumRegisters is the number of available general purpose
	// registers. The programmer should honour the same semantics
	// generally used by MIPS for such registers. R0 is always
	// zero and its value cannot be changed.
	NumRegisters = 32

	// NumStatusRegisters is the number of status registers.
	NumStatusRegisters = 4
)

// The following constants define bits in status register 0.
const (
	StatusUserMode = (1 << iota)
	StatusPaging
	StatusInterrupts
	StatusDebugStepping
	StatusDebugTracing
)

// The following constants define memory flags.
const (
	MemoryExec = (1 << iota)
	MemoryWrite
	MemoryRead
)

// The following constants define interrupt requests.
const (
	IrqHALT = iota
	IrqClock
	IrqTTY
)

// The following constants define memory mapped addresses.
const (
	MMClockFrequency = 1<<17 | iota
	MMTTYStatus
	MMTTYIn
	MMTTYOut
)

// TTY is any teletype attached to the VM.
type TTY interface {
	InterruptPending() (bool, error)
	StatusRegister() (*uint32, error)
	InRegister() (*uint32, error)
	OutRegister() (*uint32, error)
}

// VM is a virtual machine instance. The virtual machine is not
// goroutine safe; a single goroutine should manage it.
type VM struct {
	CF  uint32                     // clock frequency
	GPR [NumRegisters]uint32       // general purpose registers
	IPC uint32                     // saved program counter during interrupt
	IS0 uint32                     // saved S[0] during interrupt
	ISP uint32                     // saved GPR[29] during interrupt
	LTR time.Time                  // last time record
	M   [MemorySize]uint32         // memory
	PC  uint32                     // program counter
	S   [NumStatusRegisters]uint32 // status registers
	TTY TTY                        // terminal
}

// The following errors may be returned.
var (
	// ErrHalted indicates that the VM has been halted.
	ErrHalted = errors.New("vm: halted")

	// ErrNotPermitted indicates that a given operation is not permitted.
	ErrNotPermitted = errors.New("vm: operation not permitted")

	// ErrSIGSEGV indicates that we accessed an out of bound address.
	ErrSIGSEGV = errors.New("vm: segmentation fault")
)

// StatusDebug returns the stepping and/or tracing flags.
func (vm *VM) StatusDebug() uint32 {
	return vm.S[0] & (StatusDebugTracing | StatusDebugStepping)
}

// Memory accesses an address in memory
func (vm *VM) Memory(off uint32, flags uint32) (*uint32, error) {
	// Implement memory mapped I/O
	switch off {
	case MMClockFrequency:
		return &vm.CF, nil
	}
	if vm.TTY != nil {
		switch off {
		case MMTTYStatus:
			return vm.TTY.StatusRegister()
		case MMTTYIn:
			return vm.TTY.InRegister()
		case MMTTYOut:
			return vm.TTY.OutRegister()
		}
	}
	if (vm.S[0] & StatusPaging) != 0 {
		if (vm.S[1] & 0b11_1111_1111) != 0 {
			return nil, fmt.Errorf("%w: invalid page table base address", ErrSIGSEGV)
		}
		pageid := off >> 10
		pageoff := vm.S[1] + pageid
		if pageoff >= MemorySize {
			return nil, fmt.Errorf("%w: page entry above physical memory", ErrSIGSEGV)
		}
		pageinfo := vm.M[pageoff]
		pageflags := pageinfo & 0b111_1111
		if (pageflags & flags) != flags {
			return nil, fmt.Errorf("%w: memory flags mismatch", ErrNotPermitted)
		}
		membase := pageinfo & 0b1111_1111_1111_1111_1111_11_00_0000_0000
		memoff := off & 0b0000_0000_0000_0000_0000_00_11_1111_1111
		off = membase + memoff
		// fallthrough
	}
	if off >= MemorySize {
		return nil, ErrSIGSEGV
	}
	return &vm.M[off], nil
}

// Fetch fetches the next instruction, returns it, and increments
// the vm.PC program counter of the virtual machine.
func (vm *VM) Fetch() (uint32, error) {
	ci, err := vm.Memory(vm.PC, MemoryRead|MemoryExec)
	if err != nil {
		return 0, err
	}
	vm.PC++
	return *ci, nil
}

// String generates a string representation of the VM state.
func (vm *VM) String() string {
	s := fmt.Sprintf("{PC:%d GPR:%+v S:%+v}\n", vm.PC, vm.GPR, vm.S)
	return s
}

// DecodeOpcode decodes the opcode of an instruction.
func DecodeOpcode(ci uint32) uint32 {
	return (ci >> 27) & 0b1_1111
}

// DecodeRA decodes the first register of an instruction.
func DecodeRA(ci uint32) uint32 {
	return (ci >> 22) & 0b1_1111
}

// DecodeRB decodes the second register of an instruction.
func DecodeRB(ci uint32) uint32 {
	return (ci >> 17) & 0b1_1111
}

// DecodeRC decodes the third register of an instruction.
func DecodeRC(ci uint32) uint32 {
	return ci & 0b1_1111
}

// DecodeImm17 decodes the signed 17 bit immediate.
func DecodeImm17(ci uint32) uint32 {
	return SignExtend17(ci & 0b1_1111_1111_1111_1111)
}

// DecodeImm22 decodes the unsigned 22 bit immediate.
func DecodeImm22(ci uint32) uint32 {
	return ci & 0b11_1111_1111_1111_1111_1111
}

// Decode decodes an instruction.
func Decode(ci uint32) (opcode, ra, rb, rc, imm17, imm22 uint32) {
	return DecodeOpcode(ci), DecodeRA(ci), DecodeRB(ci), DecodeRC(ci),
		DecodeImm17(ci), DecodeImm22(ci)
}

// Interrupt executes an interrupt service routine.
func (vm *VM) Interrupt(code uint32) error {
	log.Printf("vm: irq %d", code)
	if (vm.S[2] & 0b11_1111_1111) != 0 {
		return fmt.Errorf("%w: invalid interrupt table base address", ErrSIGSEGV)
	}
	if (vm.S[3] & 0b11_1111_1111) != 0 {
		return fmt.Errorf("%w: invalid interrupt stack base address", ErrSIGSEGV)
	}
	if code >= 16 {
		code = IrqHALT // the zero handler tells the kernel to HALT
	}
	// save state and switch to interrupt
	vm.IS0 = vm.S[0]
	vm.ISP = vm.GPR[29]
	vm.IPC = vm.PC
	// swap to kernel stack
	vm.GPR[29] = vm.S[3]
	// enter kernel mode with interrupt handling and paging disabled
	vm.S[0] &^= StatusUserMode | StatusInterrupts | StatusPaging
	// jump to ISR
	off := vm.S[2] + code
	if off >= MemorySize {
		return ErrSIGSEGV
	}
	vm.PC = vm.M[off]
	return nil
}

// MaybeInterrupt checks whether there is any hardware that has
// pending interrupts and services the highest priority one.
func (vm *VM) MaybeInterrupt() error {
	if (vm.S[0] & StatusInterrupts) == 0 {
		return nil
	}
	// Clock
	if vm.CF > 0 {
		now := time.Now()
		if vm.LTR.IsZero() {
			vm.LTR = now
		}
		if now.Sub(vm.LTR).Milliseconds() >= int64(vm.CF) {
			vm.LTR = now
			return vm.Interrupt(IrqClock)
		}
		// fallthrough
	}
	// TTY
	if vm.TTY != nil {
		ok, err := vm.TTY.InterruptPending()
		if err != nil {
			return err
		}
		if ok {
			return vm.Interrupt(IrqTTY)
		}
		// fallthrough
	}
	return nil
}

// Execute executes the current instruction ci. This function returns an
// error when the processor has halted or a fault has occurred.
func (vm *VM) Execute(ci uint32) error {
	// decode instruction
	opcode, ra, rb, rc, imm17, imm22 := Decode(ci)
	// guarantee that r0 is always zero
	defer func() {
		vm.GPR[0] = 0
	}()
	// execute instruction
	switch opcode {
	case OpcodeJALR:
		// like in RiSC-16 there is no trap when either register
		// is different from zero, just a normal JALR.
		if ra != 0 || rb != 0 {
			vm.GPR[ra] = vm.PC
			vm.PC = vm.GPR[rb]
		} else if (vm.S[0] & StatusInterrupts) == 0 {
			return ErrHalted
		} else if err := vm.Interrupt(imm17); err != nil {
			return err
		}
	case OpcodeADD:
		vm.GPR[ra] = vm.GPR[rb] + vm.GPR[rc]
	case OpcodeADDI:
		vm.GPR[ra] = vm.GPR[rb] + imm17
	case OpcodeNAND:
		vm.GPR[ra] = ^(vm.GPR[rb] & vm.GPR[rc])
	case OpcodeLUI:
		vm.GPR[ra] = imm22 << 10
	case OpcodeSW, OpcodeLW:
		off := vm.GPR[rb] + imm17
		var flags uint32
		switch opcode {
		case OpcodeSW:
			flags |= MemoryWrite
		case OpcodeLW:
			flags |= MemoryRead
		}
		mptr, err := vm.Memory(off, flags)
		if err != nil {
			return err
		}
		switch opcode {
		case OpcodeSW:
			*mptr = vm.GPR[ra]
		case OpcodeLW:
			vm.GPR[ra] = *mptr
		}
	case OpcodeBEQ:
		if vm.GPR[ra] == vm.GPR[rb] {
			vm.PC += imm17
		}
	case OpcodeWSR, OpcodeRSR:
		if (vm.S[0] & StatusUserMode) != 0 {
			return ErrNotPermitted
		}
		if imm22 >= NumStatusRegisters {
			return ErrNotPermitted
		}
		switch opcode {
		case OpcodeWSR:
			vm.S[imm22] = vm.GPR[ra]
		case OpcodeRSR:
			vm.GPR[ra] = vm.S[imm22]
		}
	case OpcodeIRET:
		if (vm.S[0] & StatusUserMode) != 0 {
			return ErrNotPermitted
		}
		vm.S[0] = vm.IS0
		vm.GPR[29] = vm.ISP
		vm.PC = vm.IPC
	}
	// After the execution of each instruction, check whether we have
	// any other pending interrupt and service them.
	return vm.MaybeInterrupt()
}

// SignExtend17 extends the sign to negative values over 17 bit.
func SignExtend17(v uint32) uint32 {
	if (v & 0b00000_00000_00000_1_0000_0000_0000_0000) != 0 {
		v |= 0b11111_11111_11111_0_0000_0000_0000_0000
	}
	return v
}

// Disassemble disassembles a single instruction and returns valid
// assembly code implementing such instruction.
func Disassemble(ci uint32) string {
	// decode instruction
	opcode, ra, rb, rc, imm17, imm22 := Decode(ci)
	// disassemble instruction
	switch opcode {
	case OpcodeADD:
		return fmt.Sprintf("add r%d r%d r%d", ra, rb, rc)
	case OpcodeADDI:
		return fmt.Sprintf("addi r%d r%d %d", ra, rb, int32(imm17))
	case OpcodeNAND:
		return fmt.Sprintf("nand r%d r%d r%d", ra, rb, rc)
	case OpcodeLUI:
		return fmt.Sprintf("lui r%d %d", ra, imm22)
	case OpcodeSW:
		return fmt.Sprintf("sw r%d r%d %d", ra, rb, int32(imm17))
	case OpcodeLW:
		return fmt.Sprintf("lw r%d r%d %d", ra, rb, int32(imm17))
	case OpcodeBEQ:
		return fmt.Sprintf("beq r%d r%d %d", ra, rb, int32(imm17))
	case OpcodeJALR:
		return fmt.Sprintf("jalr r%d r%d %d", ra, rb, int32(imm17))
	case OpcodeWSR:
		return fmt.Sprintf("wsr r%d %d", ra, imm22)
	case OpcodeRSR:
		return fmt.Sprintf("rsr r%d %d", ra, imm22)
	case OpcodeIRET:
		return fmt.Sprint("iret")
	default:
		return fmt.Sprintf("<unknown instruction: %d>", ci)
	}
}

// LoadBytecode loads bytecode from the specified io.Reader and returns a
// virtual machine instance for running such bytecode.
func LoadBytecode(r io.Reader) (*VM, error) {
	vm := new(VM)
	scanner := bufio.NewScanner(r)
	var addr uint32
	for scanner.Scan() {
		line := scanner.Text()
		if index := strings.Index(line, "#"); index >= 0 {
			line = line[:index]
		}
		line = strings.TrimSpace(line)
		value, err := strconv.ParseUint(line, 0, 32)
		if err != nil {
			return nil, err
		}
		vm.M[addr] = uint32(value)
		addr++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return vm, nil
}
