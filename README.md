# goproxy

Dead simple proxy that serves SSL and proxies to & from an upstream host until either ends closes the connection.

Can be used to trivially add HTTPS in front of an HTTP server for example.
```sh
> ./goproxy --help
  -cert string
    	certificate PEM file (default "cert.pem")
  -debug
    	enable debug logging
  -key string
    	key PEM file (default "key.pem")
  -port string
    	listening port (default "443")
  -routines int
    	number of concurrent routines handling connections (default 50)
  -upstream string
    	upstream server (default "localhost:8000")
```

Available on dockerhub as uristmcdwarf/goproxy

Why? ¯\_(ツ)_/¯
