Purpose
---
`journalbeatlite` tails the systemd journal, and uploads parsed messages to elasticsearch. It has 2 key features:

1. cursors are used as document ids; this means uploads are idempotent
2. message body can be parsed if it is json, and stored as structured field

Download
---
The binary will run only on linux.

```shell
wget -Nnv https://s3.amazonaws.com/binaries-and-debs/bin/linux/journalbeatlite/journalbeatlite \
	&& chmod 0700 journalbeatlite
```

Configuration file
---
Calling `journalbeatlite -config=` will print configuration file template to stdout.

Indexing (idempotent)
---
An index will be created per day with pattern `[beat_name]-YYYY.MM.DD`, with `beat_name` specified in the config file; `beat_name` defaults to `journalbeatlite`. Messages will be routed to indexes based on their timestamp. Each message will be indexed with `id = sha256(cursor)`. This means that a messages can be safely indexed more than once without creating duplicates.

Timestamps
---
Each message's `@timestamp` field is based on `__REALTIME_TIMESTAMP` field of the journal entry.

Cursor offsets
---
The cursor of the last successfully indexed message is stored in the file specified by the `cursor_file_name` value in the config (defaults to `./cursor`). When the cursor file is present, journalbeatlite will start reading from the journal at the specified offset. When the file is absent, it will start from the beginning of the journal.

The offset file is read only at startup, so if you want to force reindexing by removing the offset file (or setting it to a different cursor) you will need to bounce the journalbeatlite.

Failfast
---
Any error will result in the program exiting. There is no retry / reconnect logic. The offset file will always store the cursor of the last successfuly indexed message. The idea is that journalbeatlite is run as a systemd service, and so you can handle backoffs / restarts there. Possible errors include: inability to read from the journal, inability to connect to elasticsearch, response code from elasticsearch other than 200 or 201.

Format
---
A typical message, as indexed into elasticsearch. 

```json
{
  "@timestamp": "2016-09-14T18:51:39.984Z",
  "beat": {
    "hostname": "journalbeatlite",
    "name": "journalbeatlite"
  },
  "journal": {
    "CODE_FILE": "../src/login/logind-session.c",
    "CODE_FUNCTION": "session_finalize",
    "CODE_LINE": "685",
    "LEADER": "28903",
    "MESSAGE_ID": "3354939424b4456d9802ca8333ed424a",
    "PRIORITY": "6",
    "SESSION_ID": "15",
    "SYSLOG_FACILITY": "4",
    "SYSLOG_IDENTIFIER": "systemd-logind",
    "USER_ID": "ubuntu",
    "_BOOT_ID": "87030f182f8747a39a977d7e1041d017",
    "_CAP_EFFECTIVE": "24420002f",
    "_CMDLINE": "/lib/systemd/systemd-logind",
    "_COMM": "systemd-logind",
    "_EXE": "/lib/systemd/systemd-logind",
    "_GID": "0",
    "_HOSTNAME": "journalbeatlite",
    "_MACHINE_ID": "11897cef820f42bd8357415b5fb4b2f9",
    "_PID": "1984",
    "_SOURCE_REALTIME_TIMESTAMP": "1473879099984448",
    "_SYSTEMD_CGROUP": "/system.slice/systemd-logind.service",
    "_SYSTEMD_SLICE": "system.slice",
    "_SYSTEMD_UNIT": "systemd-logind.service",
    "_TRANSPORT": "journal",
    "_UID": "0",
    "__CURSOR": "s=dd49c009db51435c88709fedc1dd51ca;i=689;b=87030f182f8747a39a977d7e1041d017;m=5aaa9fe8f;t=53c7c38324b2f;x=1368ac2395b42703"
  },
  "message": "Removed session 15."
}
```
