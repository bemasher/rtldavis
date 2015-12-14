#!/bin/sh
set -e

if [ ! -d "$HOME/librtlsdr/build" ]; then
	git clone https://github.com/steve-m/librtlsdr
	cd librtlsdr
	mkdir build
	cd build
	cmake ../
	make
	sudo make install
	sudo ldconfig
fi