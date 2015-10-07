FROM perl:5.20 
MAINTAINER Siddhartha Basu<siddhartha-basu@northwestern.edu>

COPY run.sh /usr/src/chadosqitch/
RUN apt-get update && apt-get -y install postgresql-client \
    && cpanm -n DBD::Pg App::Sqitch \
    && mkdir -p /usr/src/chadosqitch \
    && chmod u+x /usr/src/chadosqitch/run.sh \
    && cd /data \
    && curl -L -o sqitch-dictychado-1.23.tar.bz2 https://github.com/dictyBase/Chado-Sqitch/releases/download/dictychado-1.23/sqitch-dictychado-1.23.tar.gz \
    && tar xvjf sqitch-dictychado-1.23.tar.bz2 \
    && cd sqitch-dictychado-1.23 \
    && cp sqitch.conf /config/sqitch.conf
ENV SQITCH_CONFIG /config/sqitch.conf
CMD ["/usr/src/chadosqitch/run.sh"]

