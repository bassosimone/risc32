		addi r1 r1 4
again:	beq r1 r0 done
		addi r2 r2 100
		add r2 r2 r1
		addi r1 r1 -1
		beq r0 r0 again
done:	halt
