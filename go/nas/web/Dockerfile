FROM saichler/builder:latest AS build
COPY main.go /home/src/github.com/saichler/build/
RUN go mod init
RUN GOPROXY=direct GOPRIVATE=github.com go mod tidy
RUN go build -o nasfile

FROM saichler/nasfile-security:latest AS final
COPY --from=build /home/src/github.com/saichler/build/nasfile /home/run/nasfile
COPY web/ /home/run/web/
ENTRYPOINT ["/home/run/nasfile"]
