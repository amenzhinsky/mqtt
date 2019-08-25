fuzz:
	go-fuzz-build -tags fuzz -o mqtt-encode.zip -func FuzzEncode ./packet
	go-fuzz -bin mqtt-encode.zip -workdir testdata

fuzzclean:
	rm -rf mqtt-encode.zip testdata/{crashers,suppressions}
