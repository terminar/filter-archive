
PREFIX=/usr/local

all: filter-archive

clean:
	if [ -e ./filter-archive ] ; then rm filter-archive ; fi

filter-archive: clean
	go build

install: filter-archive
	cp -av ./filter-archive $(PREFIX)/libexec/opensmtpd
