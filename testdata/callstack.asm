        movi r29 1048575     # initialize stack ptr
        movi r1 _main        # get the address of _main
        jalr r31 r1          # call _main
        halt                 # we're done here

_main:  sw r31 r29 0
        addi r29 r29 -1      # push r31
        movi r4 17           # first argument
        movi r1 _twice       # get routine address
        jalr r31 r1          # call routine
        addi r29 r29 1
        lw r31 r29 0         # pop r31
        jalr r0 r31          # return

_twice: sw r31 r29 0
        addi r29 r29 -1      # push r31
        add r2 r4 r4         # compute result
        addi r29 r29 1
        lw r31 r29 0         # pop r31
        jalr r0 r31          # return
