again:  movi r1 48879  # 0xbeef
        nand r2 r1 r1
        addi r2 r2 1
        add r2 r2 r1
        beq r2 r0 done
        beq r0 r0 again
done:   halt
