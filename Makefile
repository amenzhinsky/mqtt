
get:
	go get -u github.com/dvyukov/go-fuzz/{go-fuzz,go-fuzz-build}

fuzz:
	go-fuzz-build -tags fuzz -o mqtt-encode.zip -func FuzzEncode .
	go-fuzz -bin mqtt-encode.zip -workdir testdata

fuzzclean:
	rm -rf mqtt-encode.zip testdata/{crashers,suppressions}
