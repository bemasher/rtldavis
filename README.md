# rtldavis
An rtl-sdr receiver for Davis Instruments weather stations.

### Purpose
This project aims to implement a receiver for Davis Instruments wireless weather stations by making use of inexpensive rtl-sdr dongles.

[![Build Status](https://travis-ci.org/bemasher/rtldavis.svg?branch=master&style=flat)](https://travis-ci.org/bemasher/rtldavis)
[![GPLv3 License](https://img.shields.io/badge/license-GPLv3-blue.svg?style=flat)](http://choosealicense.com/licenses/gpl-3.0/)

### Requirements
 * GoLang >=1.5 (Go build environment setup guide: http://golang.org/doc/code.html)
 * rtl-sdr [github.com/steve-m/librtlsdr](https://github.com/steve-m/librtlsdr)

### Building
The following instructions assume that you have already built and installed the rtl-sdr tools and library above. Please see build instructions provided here: [http://sdr.osmocom.org/trac/wiki/rtl-sdr#Buildingthesoftware](http://sdr.osmocom.org/trac/wiki/rtl-sdr#Buildingthesoftware)

To build the project, the following commands will checkout the latest source, initialize submodules and build the project:

	go get -v github.com/bemasher/rtldavis
	cd $GOPATH/github.com/bemasher/rtldavis
	git submodule init
	git submodule update
	go install -v .

This will produce the binary `$GOPATH/bin/rtldavis`. For convenience it's common to add `$GOPATH/bin` to the path.

### Usage
Available command-line flags are as follows:

```
Usage of rtldavis:
  -id int
    	id of the station to listen for
  -v	log extra information to /dev/stderr
```

### License
The source of this project is licensed under GPL v3.0. According to [http://choosealicense.com/licenses/gpl-3.0/](http://choosealicense.com/licenses/gpl-3.0/) you may:

#### Required:

 * **Disclose Source:** Source code must be made available when distributing the software. In the case of LGPL and OSL 3.0, the source for the library (and not the entire program) must be made available.
 * **License and copyright notice:** Include a copy of the license and copyright notice with the code.
 * **State Changes:** Indicate significant changes made to the code.

#### Permitted:

 * **Commercial Use:** This software and derivatives may be used for commercial purposes.
 * **Distribution:** You may distribute this software.
 * **Modification:** This software may be modified.
 * **Patent Use:** This license provides an express grant of patent rights from the contributor to the recipient.
 * **Private Use:** You may use and modify the software without distributing it.

#### Forbidden:

 * **Hold Liable:** Software is provided without warranty and the software author\/license owner cannot be held liable for damages.

### Feedback
If you have any general questions or feedback leave a comment below. For bugs, feature suggestions and anything directly relating to the program itself, submit an issue.