#!/bin/bash

echo "THIS TEST USES -race, TIMES MEASURED ARE NOT MEASURED ACCURATELY"
echo

export LOGGER_LEVEL=$1
go test -bench=. -benchtime=1s -count=1 -run=^$ -v -race ./dftp/dftp_bench_test.go
