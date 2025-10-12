#!/bin/bash

go test -bench=. -benchtime=1s -count=1 -run=^$ -v ./dftp/dftp_bench_test.go
