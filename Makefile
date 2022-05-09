healthd: main.go 
	CGO_ENABLED=0 go build -o healthd .

docker:
	docker build -t healthd .

install: healthd
	mkdir -p /opt/bin
	install -o root -g root -m 0755 healthd /opt/bin/healthd
	install -o root -g root -m 0644 healthd.service /etc/systemd/system/healthd.service
	

