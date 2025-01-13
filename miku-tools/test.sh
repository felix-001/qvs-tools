#!/bin/bash

for i in {1..1000000}; do
	go run . -cmd gb -ip 101.132.36.201
done
