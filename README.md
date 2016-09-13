Purpose
---
`journalbeatlite` tails the journal, and uploads parsed messages to elasticsearch. It has 2 key features:

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
Calling `journalbeatlite -config=` will print configuration file template to stdout. The default location for the configuration file is `./config.json`.

Indexing (idempotent)
---
An index will be created per day. Messages will be routed to indexes based on their timestamp (provided by the journal). Each message will be indexed with `id = sha256(cursor)`. This means that a messages can be safely indexed more than once without creating duplicates.

Cursor offsets
---
The cursor of the last successfully indexed message is stored in the file specified by the `"cursor_file_name"` value in the config (defaults to `./cursor`). When the cursor file is present, journalbeatlite will start reading from the journal at the specified offset. When the file is absent, it will start from the beginning of the journal. 

Metadata
---
A typical message, as indexed into elasticsearch, has a lot of juicy bits. Get excited!

```json
{
  "@timestamp": "2016-08-12T17:17:01.740Z",
  "beat": {
    "hostname": "dev",
    "name": "journalbeatlite"
  },
  "journal": {
    "PRIORITY": 6,
    "SYSLOG_FACILITY": 10,
    "SYSLOG_IDENTIFIER": "CRON",
    "SYSLOG_PID": 10658,
    "_AUDIT_LOGINUID": 0,
    "_AUDIT_SESSION": 23,
    "_BOOT_ID": "1788b7b5dba644cebfeb422b6027c379",
    "_CAP_EFFECTIVE": "3fffffffff",
    "_CMDLINE": "/usr/sbin/CRON -f",
    "_COMM": "cron",
    "_EXE": "/usr/sbin/cron",
    "_GID": 0,
    "_HOSTNAME": "dev",
    "_MACHINE_ID": "a8503c78c5a5473f86f44dff20bb348e",
    "_PID": 10658,
    "_SOURCE_REALTIME_TIMESTAMP": 1471022221740688,
    "_SYSTEMD_CGROUP": "/system.slice/cron.service",
    "_SYSTEMD_SLICE": "system.slice",
    "_SYSTEMD_UNIT": "cron.service",
    "_TRANSPORT": "syslog",
    "_UID": 0,
    "__CURSOR": "s=d7e23b1de2714e50aa0f6c387c0b6b19;i=684;b=1788b7b5dba644cebfeb422b6027c379;m=9d34c4ac5;t=539e30cfbcaea;x=9df82c74cc53b0f8"
  },
  "message": "pam_unix(cron:session): session closed for user root"
}
```
