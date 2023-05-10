FROM public.ecr.aws/lambda/provided:al2 AS build

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Get rid of the extension warning
RUN mkdir -p /opt/extensions
RUN yum -y install golang
RUN go env -w GOPROXY=direct

# Clone git, copying go.mod, go.sum, main.go
WORKDIR /var/task/
RUN yum install git -y
RUN git clone https://github.com/seedspirit/NaverCrawler-CICD-go.git
RUN cp NaverCrawler-CICD-go/main.go /var/task/
RUN cp NaverCrawler-CICD-go/go.mod /var/task/
RUN cp NaverCrawler-CICD-go/go.sum /var/task/

# cache dependencies
RUN go mod download
RUN go build -o main .

FROM public.ecr.aws/lambda/provided:al2
COPY --from=build /var/task/main /var/task/main

# Install Chrome dependencies
RUN curl https://dl.google.com/linux/direct/google-chrome-stable_current_x86_64.rpm -o chrome.rpm && \
    yum install -y ./chrome.rpm && \
    yum install -y fontconfig libX11 GConf2 dbus-x11
    
# Create a writable directory for Chrome
RUN mkdir /tmp/.com.google.Chrome
RUN chmod 777 /tmp/.com.google.Chrome

# Set the TMPDIR environment variable to /tmp
ENV TMPDIR=/tmp

ENTRYPOINT ["/var/task/main"]
