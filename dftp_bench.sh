#!/bin/bash

go test -bench=. -benchtime=1s -count=3 -run=^$ -v ./dftp/dftp_bench_test.go
