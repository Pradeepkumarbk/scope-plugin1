FROM ubuntu 

RUN apt-get -y update

RUN apt-get install -y curl

COPY ./scope-plugin1 /usr/bin/scope-plugin1

ENTRYPOINT ["/usr/bin/scope-plugin1"]