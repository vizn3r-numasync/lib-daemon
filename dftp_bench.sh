#!/bin/bash

go test -bench=. -benchtime=5s -count=5 -run=^$ -v ./dftp/dftp_bench_test.go | tee bench.txt
