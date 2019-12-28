# filter-archive

## Description
This filter implements "mail archiving" into flat files, additionally with some meta data.

The code (and even this readme) is heavily based on filter-rspamd by Gilles Chehade <gilles@poolp.org>, 
available at https://github.com/poolpOrg/filter-rspamd/.

Currently, the code is only tested on FreeBSD 12.1 with OpenSMTPD 6.6.0p1 but should work on other
systems as well.

## Features
The filter currently supports:
- writing mails (single files) into a storage folder 
  - flat into <archive-path> or 
  - with subdirectories <archive-path>/YYYY-MM/DD)
- meta-file with additional infos from smtpd session
- if write/open file errors occur, the filter will not exit (and thus should not crash opensmtpd)!
- line by line buffered writing - (not completely cached in memory)

## Dependencies
The filter is written in Golang and doesn't have any dependencies beyond standard library.

It requires OpenSMTPD 6.6.0 or higher.


## How to install
There are no available installation packages for distributions.

- install go

Clone the repository, build and install the filter:
```
$ cd filter-archive/
$ go build
$ doas install -m 0555 filter-rspamd /usr/local/libexec/smtpd/filter-archive
```

Alternatively, use the Makefile, check the PREFIX and type "make install".

## How to configure
The filter itself requires commandline parameters when called by OpenSMTPD (storage folder).
Also, the storage folder needs to be writeable by the OpenSMTPD process and should be created manually!

It must be declared in smtpd.conf and attached to a listener for sessions to write to the archive:
```
filter "archive" proc-exec "filter-archive /var/db/mail-archive"

listen on all filter "archive"
```

If the mails should be stored directly into the storage folder without subfolders, "-f" can be specified
```
filter "archive" proc-exec "filter-archive -f /var/db/mail-archive"

listen on all filter "archive"
```

