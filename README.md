# seal 🦭

Check the integrity of your file archives and backups.

## Running it locally

```bash
# creates test directory structure, see SetupTestDir
go test ./...

# run the seal command to create _seal.json files
go run ./cmd/seal ./testdir
```

## Commands

### `seal`

- Adds new files and directories to seals.
- Verifies all existing files against the seal.
- Raises errors for deleted or modified files.
- Keeps missing and modified files in the seals.
