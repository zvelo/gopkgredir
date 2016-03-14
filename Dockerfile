FROM scratch
MAINTAINER Joshua Rubin <jrubin@zvelo.com>
ADD ./gopkgredir /gopkgredir
EXPOSE 80 443
ENTRYPOINT ["/gopkgredir"]
