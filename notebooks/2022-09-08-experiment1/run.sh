#!/bin/bash

ADDR="127.0.0.1:7777"

for i in {0..256} 
do
    enbench blast -a $ADDR >> fixed_size.jsonlines
    sleep 1
done

for (( i=1; i<1025; i=i+31 ))
do
    S=$((1024*i))
    for j in {0..64}
    do
        enbench blast -a $ADDR -S $S >> variable_size.jsonlines
        sleep 1
    done
done
