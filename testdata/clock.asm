            movi r1 _boot
            jalr r0 r1
            .space 1021
__itbl:     .space 1024
__istack:   .space 2048

_boot:      nop

            # set interrupt handler base address
            movi r1 __itbl
            wsr r1 2

            # set interrupt handler for interrupt one (clock)
            movi r8 __irq1
            sw r8 r1 1

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

            .space 1234

__irq1:     nop                  # "handle" the interrupt
            rsr r29 3            # restore userspace stack
            wsr r26 0            # reset state
            jalr r0 r27          # return from interrupt