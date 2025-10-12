#!/bin/bash

echo
echo "THIS TEST USES -race, TIMES MEASURED ARE NOT ACCURATE"
echo

# Make sure test output directory exists
mkdir -p ./test

# Set log level (default: error)
export LOGGER_LEVEL=${1:-error}

# Compiler, linker, and test flags
GC_FLAGS="-N -m -l -d=ssa/check_bce/debug=1"
LD_FLAGS="-s -w"
TEST_FLAGS="-bench=. -benchtime=500ms -count=1 -run=^$ -v -race"
BENCH_FLAGS="-bench=. -benchtime=5s -count=3 -run=^$ -v"
PROF_FLAGS="-bench=. -benchtime=1s -count=1 -run=^$ -v -cpuprofile=./test/cpu.out -memprofile=./test/mem.out -trace ./test/trace.out -benchmem"

echo
echo "TESTING RACE"
echo

# Compile and run tests with profiling + race detection
go test \
  $TEST_FLAGS \
  -gcflags="all=$GC_FLAGS" \
  -ldflags="$LD_FLAGS" \
  ./dftp/dftp_bench_test.go 2>&1 | tee ./test/test.out

# Show heap escapes
echo
echo "Top heap escapes:"
grep "escapes to heap" ./test/test.out | sort | uniq -c | sort -nr | head
echo "Total heap escapes:"
grep -c "escapes to heap" ./test/test.out

echo
echo "PROFILIGING"
echo

# Run benchmarks separately (no -race, more accurate)
go test $PROF_FLAGS ./dftp/dftp_bench_test.go | tee ./test/prof.out

echo
echo "BENCHMARKING"
echo

# Run benchmarks separately (no -race, more accurate)
go test $BENCH_FLAGS ./dftp/dftp_bench_test.go | tee ./test/bench.out
