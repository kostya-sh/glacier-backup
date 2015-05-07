## Dependencies
- python >= 2.5
- boto >= 2.7

## Usage
```
glacier-backup.py DIR [-compact]
```

When *-compact* flag is specified all \*.db files in the *DIR/.glacier-backup* directory are merged into one db file.

## Configuration
A sample config file is provided: *config.sample*

User configuration (*~/.glacier_backup* file) is optional.

Directory configuration (*DIR/.glacier-backup/config* file) is required and should specify *vault* property.

## Functionality
- read information about already uploaded files from DIR/.glacier-backup/*.db files
- find new and modified files (based on file modified time)
- upload them to glacier vault from directory configuration
- update DIR/.glacier-backup/*.db files
