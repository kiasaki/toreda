SOURCES := *.go

run: toreda
	./toreda $(filter-out $@,$(MAKECMDGOALS))

toreda: $(SOURCES)
	go build -i -o toreda
