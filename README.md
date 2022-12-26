# Pstorage API Client
This an application for uploading files to pstorage from the terminal

# Usage

## With api-key flag

```bash

pstorage upload --api-key <APIKEY> files dir/files dir/*

```
## default config file
### ~/.pstorage.yaml
```yaml
api-key: <API KEY>
```
```bash
pstorage upload file dir/file dir/*
```
## custom config file
### .customfile.yaml
```yaml
api-key: <KEY>
```
```bash
pstorage upload --config .customfile.yaml  file dir/files dir/*
```

# Flags
- `--original` : print orignal url for uploaded files
- `--large` : print large url
- `--medium` : print medium url
- `--thumb` : print thumb url
