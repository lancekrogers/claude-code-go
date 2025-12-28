# Claude Code Go SDK Demo Guide

## What the Demo Does

The demo showcases Claude Code Go SDK by having Claude:

1. **Create** a Go program (`it_works/keccac.go`) that computes Keccac hashes
2. **Build** the program using `go build` or `go run`
3. **Test** it on real files to demonstrate functionality
4. **Show** working hash output

## Expected Demo Flow

1. **Run from SDK root**: `make demo` (from project top-level directory)
2. Claude explains approach (≤3 sentences)
3. User says: "yes, please start coding"
4. Claude creates `it_works/` directory
5. Claude copies `examples/demo/test-file.txt` to `it_works/test-file.txt`
6. Claude creates `it_works/keccac.go` with proper Go code
7. Claude changes to `it_works/` directory (`cd it_works/`)
8. Claude tests: `go run keccac.go test-file.txt`
9. Claude tests: `go run keccac.go ../README.md`
10. Claude shows hash output proving both files work

## Test Files Available

- `it_works/test-file.txt` - Small demo file (copied from examples/demo/)
- `../README.md` - Larger SDK documentation file (accessible from it_works/)
- Both files guaranteed to exist when demo runs from SDK root directory

## Expected Hash Output

For `it_works/test-file.txt`: `14cc70b04a44bb7cc18ee779c386549ed10058a62c7e7875ff2baf0e3ffdbd4e`

**Note**: The demo specifically uses Keccac (via `crypto/sha3.New256()`), not SHA-256. Go's `crypto/sha3` package implements the Keccac algorithm used in Ethereum and other cryptocurrencies.

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
