        movi r1 _init        # prepare to jump to _init
        jalr r0 r1           # jump there
        .space 4092          # 4092 + 3 = 4095 -- space for kernel?

_init:  sw r0 r0 0
        sw r0 r0 1
        sw r0 r0 2           # clear trampoline (movi is 2 instr + jalr)
        movi r29 1048575     # initialize stack ptr
        movi r1 _main        # get the address of _main
        jalr r31 r1          # call _main
        halt                 # we're done here

_main:  sw r31 r29 0
        addi r29 r29 -1      # push r31
        sw r0 r29 0
        addi r29 r29 -1      # push space for return value
        movi r1 4
        sw r1 r29 0
        addi r29 r29 -1      # push first argument
        movi r1 _fact        # get subroutine address
        jalr r31 r1          # call routine
        addi r29 r29 1       # pop first argument
        addi r29 r29 1
        lw r1 r29 0          # get return value
        addi r29 r29 1
        lw r31 r29 0         # pop r31
        jalr r0 r31          # return

_fact:  sw r31 r29 0
        addi r29 r29 -1      # push r31
        addi r1 r0 1
        sw r1 r29 3          # result is 1
__0:    lw r1 r29 2          # current
        beq r1 r0 __N        # we're done
        lw r2 r29 3          # result
        sw r0 r29 0
        addi r29 r29 -1      # push space for return value
        sw r1 r29 0
        addi r29 r29 -1      # push argument 1: current value
        sw r2 r29 0
        addi r29 r29 -1      # push argument 2: result
        movi r1 _mult
        jalr r31 r1          # call _mult
        addi r29 r29 1
        addi r29 r29 1
        addi r29 r29 1       # unwind
        lw r1 r29 0
        sw r1 r29 3          # update result
        lw r1 r29 2
        addi r1 r1 -1
        sw r1 r29 2          # subtract -1 to current
        beq r0 r0 __0        # retry
__N:    addi r29 r29 1
        lw r31 r29 0         # pop r31
        jalr r0 r31          # return
 
_mult:  sw r31 r29 0
        addi r29 r29 -1      # push r31
        lw r8 r29 2          # second argument
        lw r9 r29 3          # first argument
        add r10 r0 r0        # result
__mlt:  beq r8 r0 __done
        addi r8 r8 -1
        add r10 r10 r9
        beq r0 r0 __mlt      # multiply's loop
__done: sw r10 r29 4         # save on stack
        addi r29 r29 1
        lw r31 r29 0         # pop r31
        jalr r0 r31          # return
 