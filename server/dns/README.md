To use:
```
docker build -t dns-server .
docker run -dit --name dns-server -p 53:53/udp dns-server -conf /etc/coredns/Corefil
```

For the web server:
```
docker build -f Dockerfile.web -t webserver .
docker run -dit --name webserver -p 80:80 webserver
```
