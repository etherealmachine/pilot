#!/bin/sh
vulcanize --inline-scripts --inline-css --strip-comments index.html > ../index.html
vulcanize --inline-scripts --inline-css --strip-comments login.html > ../login.html
