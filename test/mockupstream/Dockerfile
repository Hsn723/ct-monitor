FROM golang:1.18.3-buster as build
ENV CGO_ENABLED=0
WORKDIR /work
COPY main.go /work
RUN go build -o mock-upstream main.go


FROM scratch
COPY --from=build /work/mock-upstream /mock-upstream
COPY *.json /
ENTRYPOINT [ "/mock-upstream" ]
