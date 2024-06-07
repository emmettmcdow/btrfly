To use:
```
docker build -t dns-server .
docker run -dit --name dns-server -p 53:53/udp dns-server -conf /etc/coredns/Corefil
```
