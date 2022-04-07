# seal 🦭

Check the integrity of your file archives and backups.

## Running it locally

```bash
# creates test directory structure, see SetupTestDir
go test ./...

# run the seal command to create _seal.json files
go run ./cmd/seal seal ./testdir
```

## Commands

### `seal [PATH...]`

- Adds new files and directories to seals.
- Verifies all existing files against the seal.
- Raises errors for deleted or modified files.
- Keeps missing and modified files in the seals.

### `verify [PATH...]`

- Checks the seal file against the current files.
- Does a quick check of just metadata first, then a second pass with hashing.
- Prints all differences in color output.
