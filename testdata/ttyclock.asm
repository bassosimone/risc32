#
# This example/test starts becoming interesting. We have a tty handler
# which echoes read characters back to the console. We also have a clock
# handler which periodically fires with tracing enabled.
#
# When this example is working as intended, the user should be able to
# see their input echoed back to the attached console. Also, the user shall
# periodically see that the clock handler is WAI thanks to tracing.
#
# This basically checks that we can have at the same time a working console
# and clock delivered interrupts to, e.g., switch process.
#
# Like other examples in the tree, we have a trampoline at the beginning
# which jumps to boot code. The interrupt stack grown downwards.
#
# Note how we're page aligning (i.e. 1<<10 aligning) a bunch of stuff since
# we need to have the interrupt table and stack aligned.
#
            movi r1 _boot
            jalr r0 r1
            .space 1021     # align to occupy a page
__itbl:     .space 1024     # one page
__istack:   .space 2048     # two pages

#
# This is the boot code of the kernel. We setup handlers and then we enter
# into an endless busy loop in which we do nothing. Since there is no power
# saving instruction, currently, we waste lots of CPU.
#
_boot:      movi r1 __itbl # set interrupt handler base address
            wsr r1 2

            # set interrupt handler for interrupt one (clock)
            movi r8 __clock
            sw r8 r1 1

            # set interrupt handler for interrupt two (tty)
            movi r8 __ttyirq
            sw r8 r1 2

            # set stack for interrupt handling
            movi r8 __istack
            wsr r8 3

            # set clock frequency to 2000 milliseconds
            addi r8 r0 2000
            movi r9 131072
            sw r8 r9 0

            # enter user mode with interrupt handling enabled
            addi r8 r0 5
            wsr r8 0

__forever:  movi r8 __forever
            jalr r0 r8

            .space 1234 # some random space before IRQ handlers

#
# This is the interrupt handler for the TTY. When an interrupt handler
# is called, the hardware has saved the state, which is going to be restored
# when we call the IRET instruction.
#
# Any register that we're using is going to be saved into the
# kernel stack and restored when leaving.
#
__ttyirq:   sw r8 r29 0          # push r8 (1/2)
            addi r29 r29 1       # push r8 (2/2)
            sw r9 r29 0          # push r9 (1/2)
            addi r29 r29 1       # push r9 (2/2)
            sw r10 r29 0         # push r10 (1/2)
            addi r29 r29 1       # push r10 (2/2)
                                 #
            movi r8 131073       # r8 = MMTTYStatus
            lw r9 r8 0           # r9 = current TTY status
            addi r10 r0 3        # r10 = TTYOut|TTYIn
            beq r8 r10 __ttydone # cannot do anything until TTYOut is gone
                                 #
            movi r8 131073       # r8 = MMTTYStatus
            lw r9 r8 0           # r9 = current TTY status
            addi r10 r0 2        # r10 = TTYOut
            beq r8 r10 __ttydone # cannot do anything until TTYOut is gone
                                 #
            movi r8 131073       # r8 = MMTTYStatus
            lw r9 r8 1           # get current input
            sw r9 r8 2           # set current output
            movi r9 2            # r9 = TTYOut
            sw r9 r8 0           # MTTYStatus = r9
                                 #
__ttydone:  addi r29 r29 -1      # pop r10 (1/2)
            lw r10 r29 0         # pop r10 (2/2)
            addi r29 r29 -1      # pop r9 (1/2)
            lw r9 r29 0          # pop r9 (2/2)
            addi r29 r29 -1      # pop r8 (1/2)
            lw r8 r29 0          # pop r8 (2/2)
                                 #
            iret                 # return from interrupt

__clock:    sw r8 r29 0          # push r8 (1/2)
            addi r29 r29 1       # push r8 (2/2)
                                 #
            rsr r8 0             #
            addi r8 r8 16        # turn on tracing
            wsr r8 0             #
                                 #
            addi r29 r29 -1      # pop r8 (1/2)
            lw r8 r29 0          # pop r8 (2/2)
                                 #
            iret                 # return from interrupt