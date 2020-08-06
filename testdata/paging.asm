            # set page table base address
            addi r1 r0 1024
            wsr r1 1

            # protect the first page as code only
            addi r8 r0 5   # 5 aka r-x
            sw r8 r1 0

            # map the second page to the tenth page and make it data
            addi r8 r0 10246  # (10<<10|6 aka rw-)
            sw r8 r1 1

            # enable paging
            addi r8 r0 2
            wsr r8 0

            # write in the first page (should cause failure)
            #sw r1 r0 1

            # access data in the second page
            addi r8 r0 17
            sw r8 r0 1024

            # jump to the second page (should cause failure)
            #beq r0 r0 1024

            # disable paging
            addi r8 r0 0
            wsr r8 0

            # clear r8
            add r8 r0 r0

            # load the previous data (should be zero b/c of paging)
            lw r8 r0 1024

            # load in the tenth page, where we should find 17
            lw r8 r0 10240