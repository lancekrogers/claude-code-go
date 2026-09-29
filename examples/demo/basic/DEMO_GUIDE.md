# Claude Code Go SDK Demo Guide

## What the Demo Does

The demo showcases Claude Code Go SDK by having Claude:

1. **Create** a Go program (`it_works/sha3sum.go`) that computes SHA3-256 hashes
2. **Build** the program using `go build` or `go run`
3. **Test** it on real files to demonstrate functionality
4. **Show** working hash output

## Expected Demo Flow

1. **Run from SDK root**: `just demo basic` (from project top-level directory)
2. Claude explains approach (≤3 sentences)
3. User says: "yes, please start coding"
4. Claude creates `it_works/` directory
5. Claude copies `examples/demo/basic/test-file.txt` to `it_works/test-file.txt`
6. Claude creates `it_works/sha3sum.go` with proper Go code
7. Claude changes to `it_works/` directory (`cd it_works/`)
8. Claude tests: `go run sha3sum.go test-file.txt`
9. Claude tests: `go run sha3sum.go ../README.md`
10. Claude shows hash output proving both files work

## Test Files Available

- `it_works/test-file.txt` - Small demo file (copied from examples/demo/basic/)
- `../README.md` - Larger SDK documentation file (accessible from it_works/)
- Both files guaranteed to exist when demo runs from SDK root directory

## Expected Hash Output

For `it_works/test-file.txt`: `5fa11c327f5e65f6f1d151ee337fca886a4d0dbdc80e4419bc475e363a8a6a85` (SHA3-256)

**Note**: The demo uses SHA3-256 from the standard library `crypto/sha3` package (Go 1.24+). SHA3-256 is not the legacy Keccak-256 used by Ethereum; the two pad input differently and produce different hashes.

## Key Benefits

- ✅ **Zero setup** - Uses Go's built-in crypto/sha3
- ✅ **Immediate results** - Files exist and work
- ✅ **Professional demo** - Shows real working code
- ✅ **Verifiable output** - Consistent hash values

## Interactive Commands

Try these after Claude creates the code:

- "Can you also build a binary version?"
- "Test it on the larger README file too"
- "Show the file contents so I can verify"
