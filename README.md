# Sync

## Current capabilities

- File transfer within the same machine or via tcp **unencrypted**.

## Usage

- same machine transfer ```sync send SRC DEST```
- over tcp  ``sync send SRC ADDRESS:DEST``

## TODOS

- Add config to the project 
    add option if directoy does not exist create it (server gets a request for a file at path P and if path does not exist and multiple directories dont then we should or not create them)
- Refactor code in server/client
- Add daemon option
- Variable block size
- support for hierarchies

- "friends" known servers that you can do exchanges with

 to add *friend* servers  `` sync add server server_name ``    