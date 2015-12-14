#!/bin/sh
set -e

if [ ! -d "$HOME/librtlsdr/build" ]; then
	git clone https://github.com/steve-m/librtlsdr
	cd librtlsdr
	mkdir build
	cd build
	cmake -DCMAKE_INSTALL_PREFIX:PATH=$HOME/librtlsdr/build ../
	make
	make install
	ldconfig
fi