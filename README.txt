Dependencies
------------
python 2.5
boto 2.6-dev (not 2.6.0)

Usage
-----
glacier-backup.py directory [-compact]

if -compact flag is specified then all *.db files in the config directory will be
merged together (duplicates are removed) into one db file.

Configuration
-------------
User configuration: ~/.glacier_backup
Directory configuration .glacier-backup/config

Directory configuration is required and should specify "vault" property.

Functionality
-------------
- read information about already uploaded files from DIR/.glacier-backup/*.db files
- find new and modified files (based on file modified time)
- upload them to glacier vault from directory configuration
- update DIR/.glacier-backup/*.db files
