 # camera_checker_server
 This is an IEF application
 ## How to use
 
 ### clone git repository
 ```$xslt
git clone git@github.com:MecaCho/rtsp_server.git
```
 
 
 ### build go binary file
 ```$xslt
 cd rtsp_server/cmd
 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o camera_checker
 cp cmd/camera_checker ./
```

 
 ### build docker images
 ```$xslt
docker build -t xxx:xxx .
```
 
 
 ## 
