again:  movi r4 17
        movi r1 twice
        jalr r31 r1
        movi r1 34
        beq r1 r2 done
        beq r0 r0 again

done:   halt

twice:  add r2 r4 r4
        jalr r0 r31