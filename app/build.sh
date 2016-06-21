#!/bin/sh
vulcanize \
	--abspath . \
	--inline-scripts \
	--inline-css \
	--strip-comments \
	index.html > ../index.html
vulcanize \
	--abspath . \
	--inline-scripts \
	--inline-css \
	--strip-comments \
	login.html > ../login.html
